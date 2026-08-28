package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// idempotencyKeyConstraint is the unique index from migration 000007 —
// checked by name so a duplicate-key race is distinguished from any other
// unique-violation the sales table might one day acquire.
const idempotencyKeyConstraint = "idx_sales_idempotency_key"

// SaleRepository adapts the sqlc-generated Queries to domain.SaleRepository.
// Like InventoryRepository, it holds the pool directly so Create can run
// every step of a sale — stock verification, the Sale/SaleItem inserts, the
// SALE movements, and the per-product stock decrement — inside one
// transaction.
type SaleRepository struct {
	pool *pgxpool.Pool
	q    *Queries
}

func NewSaleRepository(pool *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{pool: pool, q: New(pool)}
}

// Create verifies every item against live stock, then persists the sale
// atomically. Quantities for repeated product_id lines within the same sale
// are aggregated before the stock check and the decrement, so e.g. two
// 3-unit lines for a product with 5 in stock correctly fail as insufficient
// stock (6 > 5) instead of each line passing a stale independent check.
//
// When sale.IdempotencyKey is set, Create first checks whether a sale with
// that key already exists (active or soft-deleted) and, if so, returns it
// unchanged instead of processing anything — see domain.SaleRepository.
func (r *SaleRepository) Create(ctx context.Context, sale *domain.Sale, createdBy uuid.UUID) (bool, error) {
	if len(sale.Items) == 0 {
		return false, domain.ErrEmptySale
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

	if sale.IdempotencyKey != nil {
		existingRow, err := qtx.GetSaleByIdempotencyKey(ctx, toNullUUID(sale.IdempotencyKey))
		if err == nil {
			if err := loadSaleInto(ctx, qtx, sale, existingRow); err != nil {
				return false, err
			}
			return true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
	}

	var order []uuid.UUID
	required := map[uuid.UUID]int32{}
	for _, item := range sale.Items {
		if _, seen := required[item.ProductID]; !seen {
			order = append(order, item.ProductID)
		}
		required[item.ProductID] += item.Quantity
	}

	products := map[uuid.UUID]Product{}
	var notFound, insufficient []string
	for _, id := range order {
		p, err := qtx.GetProductByID(ctx, toUUID(id))
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				notFound = append(notFound, id.String())
				continue
			}
			return false, err
		}
		if p.CurrentStock < required[id] {
			insufficient = append(insufficient, fmt.Sprintf("%s (requested %d, available %d)", p.Name, required[id], p.CurrentStock))
			continue
		}
		products[id] = p
	}
	if len(notFound) > 0 {
		return false, fmt.Errorf("%w: products %v", domain.ErrProductNotFound, notFound)
	}
	if len(insufficient) > 0 {
		return false, fmt.Errorf("%w: %v", domain.ErrInsufficientStock, insufficient)
	}

	saleRow, err := qtx.CreateSale(ctx, CreateSaleParams{
		TotalCents:     sale.TotalCents,
		PaymentMethod:  sale.PaymentMethod,
		CreatedBy:      toNullUUID(&createdBy),
		IdempotencyKey: toNullUUID(sale.IdempotencyKey),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if sale.IdempotencyKey != nil && errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode && pgErr.ConstraintName == idempotencyKeyConstraint {
			// Lost the race against a concurrent request carrying the same
			// key: this transaction is now aborted by Postgres, so recover
			// the winner's row with a fresh query outside of it instead of
			// propagating the constraint error.
			existingRow, getErr := r.q.GetSaleByIdempotencyKey(ctx, toNullUUID(sale.IdempotencyKey))
			if getErr != nil {
				return false, getErr
			}
			if loadErr := loadSaleInto(ctx, r.q, sale, existingRow); loadErr != nil {
				return false, loadErr
			}
			return true, nil
		}
		return false, err
	}

	items := make([]domain.SaleItem, 0, len(sale.Items))
	for _, item := range sale.Items {
		itemRow, err := qtx.CreateSaleItem(ctx, CreateSaleItemParams{
			SaleID:         saleRow.ID,
			ProductID:      toUUID(item.ProductID),
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		})
		if err != nil {
			return false, err
		}
		items = append(items, *toDomainSaleItem(itemRow))

		if _, err := qtx.CreateInventoryMovement(ctx, CreateInventoryMovementParams{
			ProductID: toUUID(item.ProductID),
			Type:      domain.MovementTypeSale,
			Quantity:  item.Quantity,
			CreatedBy: toNullUUID(&createdBy),
		}); err != nil {
			return false, err
		}
	}

	for _, id := range order {
		if _, err := qtx.DecrementProductStockForSale(ctx, DecrementProductStockForSaleParams{
			Quantity: required[id],
			ID:       toUUID(id),
			Version:  products[id].Version,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return false, domain.ErrOptimisticLockConflict
			}
			return false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	*sale = *toDomainSale(saleRow)
	sale.Items = items
	return false, nil
}

// loadSaleInto populates sale (and its Items) from an already-fetched sales
// row, using q for the item lookup — q may be a plain *Queries or one bound
// to an in-flight transaction (WithTx), since both share the same type.
func loadSaleInto(ctx context.Context, q *Queries, sale *domain.Sale, row Sale) error {
	itemRows, err := q.ListSaleItemsBySaleID(ctx, row.ID)
	if err != nil {
		return err
	}
	*sale = *toDomainSale(row)
	sale.Items = make([]domain.SaleItem, len(itemRows))
	for i, itemRow := range itemRows {
		sale.Items[i] = *toDomainSaleItem(itemRow)
	}
	return nil
}

func (r *SaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	row, err := r.q.GetSaleByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSaleNotFound
		}
		return nil, err
	}
	itemRows, err := r.q.ListSaleItemsBySaleID(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	sale := toDomainSale(row)
	sale.Items = make([]domain.SaleItem, len(itemRows))
	for i, itemRow := range itemRows {
		sale.Items[i] = *toDomainSaleItem(itemRow)
	}
	return sale, nil
}

func (r *SaleRepository) List(ctx context.Context, params domain.ListSalesParams) ([]domain.Sale, error) {
	rows, err := r.q.ListSales(ctx, ListSalesParams{
		FromDate: toTimestamptz(params.From),
		ToDate:   toTimestamptz(params.To),
		Limit:    params.Limit,
		Offset:   params.Offset,
	})
	if err != nil {
		return nil, err
	}
	sales := make([]domain.Sale, len(rows))
	for i, row := range rows {
		sales[i] = *toDomainSale(row)
	}
	return sales, nil
}

func (r *SaleRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	rowsAffected, err := r.q.SoftDeleteSale(ctx, SoftDeleteSaleParams{
		ID:        toUUID(id),
		DeletedBy: toNullUUID(&deletedBy),
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrSaleNotFound
	}
	return nil
}

// Restore clears deleted_at/deleted_by only. It never reverses the original
// stock decrement or recreates SALE movements — those already happened
// atomically inside Create. Like InventoryRepository.Restore, this just
// brings a historical record back into view for audit purposes.
func (r *SaleRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	row, err := r.q.RestoreSale(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSaleNotFound
		}
		return nil, err
	}
	return toDomainSale(row), nil
}

func (r *SaleRepository) ListTrash(ctx context.Context) ([]domain.TrashedSale, error) {
	rows, err := r.q.ListTrashedSales(ctx)
	if err != nil {
		return nil, err
	}
	trashed := make([]domain.TrashedSale, len(rows))
	for i, row := range rows {
		trashed[i] = domain.TrashedSale{
			Sale: domain.Sale{
				ID:            fromUUID(row.ID),
				TotalCents:    row.TotalCents,
				PaymentMethod: row.PaymentMethod,
				CreatedAt:     fromTimestamptz(row.CreatedAt),
				CreatedBy:     fromNullUUID(row.CreatedBy),
				DeletedAt:     fromNullTimestamptz(row.DeletedAt),
				DeletedBy:     fromNullUUID(row.DeletedBy),
			},
			DeletedByEmail: fromText(row.DeletedByEmail),
		}
	}
	return trashed, nil
}

func toDomainSale(row Sale) *domain.Sale {
	return &domain.Sale{
		ID:             fromUUID(row.ID),
		IdempotencyKey: fromNullUUID(row.IdempotencyKey),
		TotalCents:     row.TotalCents,
		PaymentMethod:  row.PaymentMethod,
		CreatedAt:      fromTimestamptz(row.CreatedAt),
		CreatedBy:      fromNullUUID(row.CreatedBy),
		DeletedAt:      fromNullTimestamptz(row.DeletedAt),
		DeletedBy:      fromNullUUID(row.DeletedBy),
	}
}

func toDomainSaleItem(row SaleItem) *domain.SaleItem {
	return &domain.SaleItem{
		ID:             fromUUID(row.ID),
		SaleID:         fromUUID(row.SaleID),
		ProductID:      fromUUID(row.ProductID),
		Quantity:       row.Quantity,
		UnitPriceCents: row.UnitPriceCents,
		CreatedAt:      fromTimestamptz(row.CreatedAt),
	}
}
