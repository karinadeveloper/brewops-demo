package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

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
func (r *SaleRepository) Create(ctx context.Context, sale *domain.Sale, createdBy uuid.UUID) error {
	if len(sale.Items) == 0 {
		return domain.ErrEmptySale
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)

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
			return err
		}
		if p.CurrentStock < required[id] {
			insufficient = append(insufficient, fmt.Sprintf("%s (requested %d, available %d)", p.Name, required[id], p.CurrentStock))
			continue
		}
		products[id] = p
	}
	if len(notFound) > 0 {
		return fmt.Errorf("%w: products %v", domain.ErrProductNotFound, notFound)
	}
	if len(insufficient) > 0 {
		return fmt.Errorf("%w: %v", domain.ErrInsufficientStock, insufficient)
	}

	saleRow, err := qtx.CreateSale(ctx, CreateSaleParams{
		TotalCents:    sale.TotalCents,
		PaymentMethod: sale.PaymentMethod,
		CreatedBy:     toNullUUID(&createdBy),
	})
	if err != nil {
		return err
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
			return err
		}
		items = append(items, *toDomainSaleItem(itemRow))

		if _, err := qtx.CreateInventoryMovement(ctx, CreateInventoryMovementParams{
			ProductID: toUUID(item.ProductID),
			Type:      domain.MovementTypeSale,
			Quantity:  item.Quantity,
			CreatedBy: toNullUUID(&createdBy),
		}); err != nil {
			return err
		}
	}

	for _, id := range order {
		if _, err := qtx.DecrementProductStockForSale(ctx, DecrementProductStockForSaleParams{
			Quantity: required[id],
			ID:       toUUID(id),
			Version:  products[id].Version,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrOptimisticLockConflict
			}
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	*sale = *toDomainSale(saleRow)
	sale.Items = items
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
		ID:            fromUUID(row.ID),
		TotalCents:    row.TotalCents,
		PaymentMethod: row.PaymentMethod,
		CreatedAt:     fromTimestamptz(row.CreatedAt),
		CreatedBy:     fromNullUUID(row.CreatedBy),
		DeletedAt:     fromNullTimestamptz(row.DeletedAt),
		DeletedBy:     fromNullUUID(row.DeletedBy),
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
