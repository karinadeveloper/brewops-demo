//go:build integration

// Package repository_test contains integration tests that exercise the real
// Postgres database (via DATABASE_URL) instead of mocks — the only way to
// prove the transactional guarantees (atomic rollback, real optimistic
// concurrency under Postgres row locking, live low-stock data) that a
// mock-based service test can't demonstrate on its own. Run with:
//
//	go test -tags=integration ./internal/repository/...
//
// Requires a reachable Postgres at DATABASE_URL with migrations applied:
//
//	docker-compose up -d postgres
//	migrate -path db/migrations -database "$DATABASE_URL" up
package repository_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/repository"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// createTestUser satisfies the FK from products/sales/inventory_movements
// to users(id) — every write in these tests needs a real user row.
func createTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	users := repository.NewUserRepository(pool)
	u := &domain.User{
		Email:        fmt.Sprintf("integration-%s@brewops.mx", uuid.NewString()),
		PasswordHash: "not-a-real-hash",
		Role:         "ADMIN",
	}
	if err := users.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return u.ID
}

func createTestProduct(t *testing.T, products *service.ProductService, createdBy uuid.UUID, stock, minStock int32) *domain.Product {
	t.Helper()
	p, err := products.Create(context.Background(), service.CreateProductInput{
		Name:           "Integration Test Product " + uuid.NewString(),
		Category:       "juice",
		SalePriceCents: 1000,
		CostCents:      500,
		CurrentStock:   stock,
		MinStock:       minStock,
		CreatedBy:      createdBy,
	})
	if err != nil {
		t.Fatalf("failed to create test product: %v", err)
	}
	return p
}

func containsProduct(products []domain.Product, id uuid.UUID) bool {
	for _, p := range products {
		if p.ID == id {
			return true
		}
	}
	return false
}

// TestLowStockFlow_SaleReducesStockToOrBelowMinStock_AppearsInLowStock covers
// the low-stock alert requirement end-to-end: a real sale, through the real
// stock-decrement path, must be reflected immediately by
// GET /products/low-stock.
func TestLowStockFlow_SaleReducesStockToOrBelowMinStock_AppearsInLowStock(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))

	product := createTestProduct(t, products, userID, 10, 8) // stock=10, min_stock=8: not low yet

	before, err := products.ListLowStock(context.Background())
	if err != nil {
		t.Fatalf("failed to list low-stock products: %v", err)
	}
	if containsProduct(before, product.ID) {
		t.Fatalf("product %v should not be low-stock before the sale", product.ID)
	}

	// Act — sell 5 units, leaving current_stock=5 <= min_stock=8.
	_, _, err = sales.Create(context.Background(), service.CreateSaleInput{
		Items:         []domain.SaleItemInput{{ProductID: product.ID, Quantity: 5, UnitPriceCents: 1500}},
		PaymentMethod: "CASH",
		CreatedBy:     userID,
	})
	if err != nil {
		t.Fatalf("expected sale to succeed, got %v", err)
	}

	// Assert
	after, err := products.ListLowStock(context.Background())
	if err != nil {
		t.Fatalf("failed to list low-stock products: %v", err)
	}
	if !containsProduct(after, product.ID) {
		t.Fatalf("expected product %v to appear in low-stock after the sale", product.ID)
	}

	updated, err := products.Get(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if updated.CurrentStock != 5 {
		t.Fatalf("expected current_stock=5 after selling 5 of 10, got %d", updated.CurrentStock)
	}
}

// TestSaleCreate_InsufficientStockItem_RollsBackEverything proves the
// atomicity a multi-item sale requires: when one item fails its stock
// check, NOTHING is persisted — not the sale, not any sale_item,
// not any inventory_movement, and no product's current_stock changes,
// including the OTHER item that had enough stock on its own. A mock-based
// service test can assert the returned error, but only a real transactional
// rollback proves this.
func TestSaleCreate_InsufficientStockItem_RollsBackEverything(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))

	okProduct := createTestProduct(t, products, userID, 50, 5)
	shortProduct := createTestProduct(t, products, userID, 3, 1)

	// Act
	_, _, err := sales.Create(context.Background(), service.CreateSaleInput{
		Items: []domain.SaleItemInput{
			{ProductID: okProduct.ID, Quantity: 10, UnitPriceCents: 1000},
			{ProductID: shortProduct.ID, Quantity: 100, UnitPriceCents: 1000}, // only 3 in stock
		},
		PaymentMethod: "CASH",
		CreatedBy:     userID,
	})

	// Assert
	if err == nil || !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}

	reloadedOK, err := products.Get(context.Background(), okProduct.ID)
	if err != nil {
		t.Fatalf("failed to reload ok product: %v", err)
	}
	if reloadedOK.CurrentStock != 50 {
		t.Fatalf("expected untouched product's stock to remain 50, got %d (stock was decremented despite the rollback)", reloadedOK.CurrentStock)
	}

	reloadedShort, err := products.Get(context.Background(), shortProduct.ID)
	if err != nil {
		t.Fatalf("failed to reload short-stock product: %v", err)
	}
	if reloadedShort.CurrentStock != 3 {
		t.Fatalf("expected short-stock product's stock to remain 3, got %d", reloadedShort.CurrentStock)
	}

	ctx := context.Background()
	var saleCount, itemCount, movementCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sales WHERE created_by = $1`, userID).Scan(&saleCount); err != nil {
		t.Fatalf("failed to count sales: %v", err)
	}
	if saleCount != 0 {
		t.Fatalf("expected 0 sales rows for this user, got %d — rollback did not happen", saleCount)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sale_items WHERE product_id IN ($1, $2)`, okProduct.ID, shortProduct.ID).Scan(&itemCount); err != nil {
		t.Fatalf("failed to count sale_items: %v", err)
	}
	if itemCount != 0 {
		t.Fatalf("expected 0 sale_items rows, got %d — rollback did not happen", itemCount)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inventory_movements WHERE product_id IN ($1, $2) AND type = 'SALE'`, okProduct.ID, shortProduct.ID).Scan(&movementCount); err != nil {
		t.Fatalf("failed to count inventory_movements: %v", err)
	}
	if movementCount != 0 {
		t.Fatalf("expected 0 SALE inventory_movements rows, got %d — rollback did not happen", movementCount)
	}
}

// TestSaleCreate_ProductVersionChangesMidTransaction_ReturnsOptimisticLockConflict
// deterministically forces the exact concurrent-edit race the
// optimistic-concurrency version column exists to catch — no timing luck
// involved. A raw transaction reads the product's version and applies
// (but does not yet commit) its own stock decrement, exactly like the first
// half of SaleRepository.Create's version-guarded update. A concurrent real
// sale for the same product reads the same still-current version (Postgres
// read-committed semantics: it can't see the other transaction's uncommitted
// write), passes its own check, and then blocks on Postgres's row lock when
// it reaches its own version-guarded UPDATE. Once the raw transaction
// commits, the sale's blocked UPDATE resumes, finds the version already
// moved on, and the whole sale must roll back with ErrOptimisticLockConflict
// — never silently retried, never allowed to oversell.
func TestSaleCreate_ProductVersionChangesMidTransaction_ReturnsOptimisticLockConflict(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))

	product := createTestProduct(t, products, userID, 20, 0)
	productPgID := pgtype.UUID{Bytes: product.ID, Valid: true}

	ctx := context.Background()
	blockerTx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin blocking transaction: %v", err)
	}
	blockerQ := repository.New(blockerTx)

	current, err := blockerQ.GetProductByID(ctx, productPgID)
	if err != nil {
		t.Fatalf("failed to read product inside blocking transaction: %v", err)
	}
	if _, err := blockerQ.DecrementProductStockForSale(ctx, repository.DecrementProductStockForSaleParams{
		Quantity: 1,
		ID:       productPgID,
		Version:  current.Version,
	}); err != nil {
		t.Fatalf("failed to apply blocking decrement: %v", err)
	}

	// Act — the real sale runs concurrently; its internal version-guarded
	// UPDATE for the same row blocks on blockerTx's still-open row lock
	// until we commit it below.
	resultCh := make(chan error, 1)
	go func() {
		_, _, err := sales.Create(context.Background(), service.CreateSaleInput{
			Items:         []domain.SaleItemInput{{ProductID: product.ID, Quantity: 1, UnitPriceCents: 1000}},
			PaymentMethod: "CASH",
			CreatedBy:     userID,
		})
		resultCh <- err
	}()

	// Give the goroutine time to reach and block on its own decrement.
	time.Sleep(500 * time.Millisecond)
	if err := blockerTx.Commit(ctx); err != nil {
		t.Fatalf("failed to commit blocking transaction: %v", err)
	}

	// Assert
	select {
	case err := <-resultCh:
		if err == nil || !errors.Is(err, domain.ErrOptimisticLockConflict) {
			t.Fatalf("expected ErrOptimisticLockConflict, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for concurrent sale to complete")
	}

	reloaded, err := products.Get(ctx, product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.CurrentStock != 19 {
		t.Fatalf("expected only the blocking transaction's decrement to have applied (19), got %d — the conflicting sale was not fully rolled back", reloaded.CurrentStock)
	}
}

// TestSaleCreate_NewIdempotencyKey_CreatesSaleNormally proves an unseen
// idempotency_key doesn't change the ordinary creation path at all.
func TestSaleCreate_NewIdempotencyKey_CreatesSaleNormally(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))
	product := createTestProduct(t, products, userID, 10, 0)
	key := uuid.New()

	// Act
	sale, existed, err := sales.Create(context.Background(), service.CreateSaleInput{
		Items:          []domain.SaleItemInput{{ProductID: product.ID, Quantity: 2, UnitPriceCents: 1500}},
		PaymentMethod:  "CASH",
		CreatedBy:      userID,
		IdempotencyKey: &key,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if existed {
		t.Fatal("expected existed=false for a brand-new idempotency key")
	}
	reloaded, err := products.Get(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.CurrentStock != 8 {
		t.Fatalf("expected current_stock=8 after selling 2 of 10, got %d", reloaded.CurrentStock)
	}
	if sale.ID == uuid.Nil {
		t.Fatal("expected a persisted sale id")
	}
}

// TestSaleCreate_RepeatedIdempotencyKey_ReturnsExistingSaleWithoutDuplicateRowOrExtraStockDecrement
// is the core guarantee this patch exists for: a retried POST /sales with
// the same idempotency_key must not create a second sale row or decrement
// stock a second time — it must return the original sale.
func TestSaleCreate_RepeatedIdempotencyKey_ReturnsExistingSaleWithoutDuplicateRowOrExtraStockDecrement(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))
	product := createTestProduct(t, products, userID, 10, 0)
	key := uuid.New()
	input := service.CreateSaleInput{
		Items:          []domain.SaleItemInput{{ProductID: product.ID, Quantity: 3, UnitPriceCents: 1500}},
		PaymentMethod:  "CASH",
		CreatedBy:      userID,
		IdempotencyKey: &key,
	}

	// Act — the same request, sent twice, exactly like a client retry after
	// a lost response.
	first, firstExisted, err := sales.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("expected first call to succeed, got %v", err)
	}
	second, secondExisted, err := sales.Create(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("expected the retried call to succeed as a no-op, got %v", err)
	}
	if firstExisted {
		t.Fatal("expected the first call to report existed=false")
	}
	if !secondExisted {
		t.Fatal("expected the retried call to report existed=true")
	}
	if second.ID != first.ID {
		t.Fatalf("expected the retried call to return the original sale %v, got %v", first.ID, second.ID)
	}

	var saleCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM sales WHERE idempotency_key = $1`, key).Scan(&saleCount); err != nil {
		t.Fatalf("failed to count sales by idempotency_key: %v", err)
	}
	if saleCount != 1 {
		t.Fatalf("expected exactly 1 sale row for this idempotency key, got %d", saleCount)
	}

	reloaded, err := products.Get(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.CurrentStock != 7 {
		t.Fatalf("expected current_stock=7 (decremented only once, 10-3), got %d", reloaded.CurrentStock)
	}
}

// TestSaleCreate_ConcurrentSameIdempotencyKey_OnlyOneInsertsTheOtherRecoversExistingSale
// covers the race the idempotency key exists to guard against: two requests
// carrying the same key arriving almost simultaneously (a real risk in the
// offline sync flow, where a response can be lost after the server already
// committed). The unique index is the final arbiter —
// exactly one goroutine's INSERT succeeds, and the loser's unique-violation
// is caught and turned into a lookup of the winner's row, never a
// propagated DB error.
func TestSaleCreate_ConcurrentSameIdempotencyKey_OnlyOneInsertsTheOtherRecoversExistingSale(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))
	product := createTestProduct(t, products, userID, 100, 0)
	key := uuid.New()
	input := service.CreateSaleInput{
		Items:          []domain.SaleItemInput{{ProductID: product.ID, Quantity: 1, UnitPriceCents: 1000}},
		PaymentMethod:  "CASH",
		CreatedBy:      userID,
		IdempotencyKey: &key,
	}

	// Act — fire both requests at once.
	var wg sync.WaitGroup
	results := make([]*domain.Sale, 2)
	existedFlags := make([]bool, 2)
	errs := make([]error, 2)
	wg.Add(2)
	for i := range 2 {
		go func(i int) {
			defer wg.Done()
			results[i], existedFlags[i], errs[i] = sales.Create(context.Background(), input)
		}(i)
	}
	wg.Wait()

	// Assert
	for i, err := range errs {
		if err != nil {
			t.Fatalf("expected no error from goroutine %d, got %v", i, err)
		}
	}
	if results[0].ID != results[1].ID {
		t.Fatalf("expected both goroutines to agree on one sale, got %v and %v", results[0].ID, results[1].ID)
	}
	if existedFlags[0] == existedFlags[1] {
		t.Fatalf("expected exactly one goroutine to win (existed=false) and the other to lose (existed=true), got %v and %v", existedFlags[0], existedFlags[1])
	}

	var saleCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM sales WHERE idempotency_key = $1`, key).Scan(&saleCount); err != nil {
		t.Fatalf("failed to count sales by idempotency_key: %v", err)
	}
	if saleCount != 1 {
		t.Fatalf("expected exactly 1 sale row despite the concurrent attempts, got %d", saleCount)
	}

	reloaded, err := products.Get(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.CurrentStock != 99 {
		t.Fatalf("expected current_stock=99 (decremented exactly once, 100-1), got %d", reloaded.CurrentStock)
	}
}

// TestSaleCreate_NoIdempotencyKey_BehavesExactlyAsBefore proves compatibility
// with any caller that doesn't send an idempotency_key: each call is an
// independent sale, just like before this patch.
func TestSaleCreate_NoIdempotencyKey_BehavesExactlyAsBefore(t *testing.T) {
	// Arrange
	pool := testPool(t)
	userID := createTestUser(t, pool)
	products := service.NewProductService(repository.NewProductRepository(pool))
	sales := service.NewSaleService(repository.NewSaleRepository(pool))
	product := createTestProduct(t, products, userID, 10, 0)
	input := service.CreateSaleInput{
		Items:         []domain.SaleItemInput{{ProductID: product.ID, Quantity: 1, UnitPriceCents: 1000}},
		PaymentMethod: "CASH",
		CreatedBy:     userID,
	}

	// Act
	first, firstExisted, err := sales.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("expected first call to succeed, got %v", err)
	}
	second, secondExisted, err := sales.Create(context.Background(), input)

	// Assert
	if err != nil {
		t.Fatalf("expected second call to succeed, got %v", err)
	}
	if firstExisted || secondExisted {
		t.Fatal("expected existed=false for both calls when no idempotency_key is sent")
	}
	if first.ID == second.ID {
		t.Fatal("expected two independent sales, not the same one, when no idempotency_key is sent")
	}
	reloaded, err := products.Get(context.Background(), product.ID)
	if err != nil {
		t.Fatalf("failed to reload product: %v", err)
	}
	if reloaded.CurrentStock != 8 {
		t.Fatalf("expected current_stock=8 after two separate 1-unit sales (10-1-1), got %d", reloaded.CurrentStock)
	}
}
