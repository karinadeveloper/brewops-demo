package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// InventoryRepository adapts the sqlc-generated Queries to
// domain.InventoryMovementRepository. Unlike ProductRepository, it also
// holds the pool directly: CreateManual needs to run its insert and its
// product stock update inside one DB transaction, which the shared
// package-level Queries (bound to the pool, not a transaction) can't do on
// its own.
type InventoryRepository struct {
	pool *pgxpool.Pool
	q    *Queries
}

func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool, q: New(pool)}
}

// CreateManual inserts m and applies stockDelta to the product's
// current_stock in a single transaction. The AdjustProductStock query is
// guarded (WHERE deleted_at IS NULL AND current_stock+delta >= 0) so it
// naturally returns zero rows for either a missing/soft-deleted product or
// an insufficient-stock delta; a follow-up GetProductByID (still inside the
// same transaction) tells those two cases apart for the caller.
func (r *InventoryRepository) CreateManual(ctx context.Context, m *domain.InventoryMovement, stockDelta int32) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := r.q.WithTx(tx)
	productID := toUUID(m.ProductID)

	if _, err := qtx.AdjustProductStock(ctx, AdjustProductStockParams{
		Delta: stockDelta,
		ID:    productID,
	}); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, getErr := qtx.GetProductByID(ctx, productID); errors.Is(getErr, pgx.ErrNoRows) {
			return domain.ErrProductNotFound
		}
		return domain.ErrInsufficientStock
	}

	row, err := qtx.CreateInventoryMovement(ctx, CreateInventoryMovementParams{
		ProductID: productID,
		Type:      m.Type,
		Quantity:  m.Quantity,
		Reason:    toText(m.Reason),
		CreatedBy: toNullUUID(m.CreatedBy),
	})
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	*m = *toDomainInventoryMovement(row)
	return nil
}

func (r *InventoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	row, err := r.q.GetInventoryMovementByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMovementNotFound
		}
		return nil, err
	}
	return toDomainInventoryMovement(row), nil
}

func (r *InventoryRepository) GetTrashedByID(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	row, err := r.q.GetTrashedInventoryMovementByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMovementNotFound
		}
		return nil, err
	}
	return toDomainInventoryMovement(row), nil
}

func (r *InventoryRepository) List(ctx context.Context, params domain.ListMovementsParams) ([]domain.InventoryMovement, error) {
	rows, err := r.q.ListInventoryMovements(ctx, ListInventoryMovementsParams{
		ProductID: toNullUUID(params.ProductID),
		FromDate:  toTimestamptz(params.From),
		ToDate:    toTimestamptz(params.To),
		Limit:     params.Limit,
		Offset:    params.Offset,
	})
	if err != nil {
		return nil, err
	}
	movements := make([]domain.InventoryMovement, len(rows))
	for i, row := range rows {
		movements[i] = *toDomainInventoryMovement(row)
	}
	return movements, nil
}

func (r *InventoryRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	rowsAffected, err := r.q.SoftDeleteInventoryMovement(ctx, SoftDeleteInventoryMovementParams{
		ID:        toUUID(id),
		DeletedBy: toNullUUID(&deletedBy),
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrMovementNotFound
	}
	return nil
}

// Restore clears deleted_at/deleted_by only. It deliberately never touches
// current_stock — a manual movement's effect on stock already happened
// (atomically, in CreateManual) at the moment it was created, not at the
// moment it's viewed. Restoring a soft-deleted movement just makes a
// historical record visible again for audit purposes; it is not a request
// to re-run the movement. This mirrors service.InventoryService.Restore and
// is the opposite of domain.Product.Restore, which does revalidate a live
// invariant (current_stock >= 0) because a product's active state, unlike a
// movement's history, can still be wrong.
func (r *InventoryRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	row, err := r.q.RestoreInventoryMovement(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMovementNotFound
		}
		return nil, err
	}
	return toDomainInventoryMovement(row), nil
}

func (r *InventoryRepository) ListTrash(ctx context.Context) ([]domain.TrashedInventoryMovement, error) {
	rows, err := r.q.ListTrashedInventoryMovements(ctx)
	if err != nil {
		return nil, err
	}
	trashed := make([]domain.TrashedInventoryMovement, len(rows))
	for i, row := range rows {
		trashed[i] = domain.TrashedInventoryMovement{
			InventoryMovement: domain.InventoryMovement{
				ID:        fromUUID(row.ID),
				ProductID: fromUUID(row.ProductID),
				Type:      row.Type,
				Quantity:  row.Quantity,
				Reason:    fromText(row.Reason),
				CreatedAt: fromTimestamptz(row.CreatedAt),
				CreatedBy: fromNullUUID(row.CreatedBy),
				DeletedAt: fromNullTimestamptz(row.DeletedAt),
				DeletedBy: fromNullUUID(row.DeletedBy),
			},
			DeletedByEmail: fromText(row.DeletedByEmail),
		}
	}
	return trashed, nil
}

func toDomainInventoryMovement(row InventoryMovement) *domain.InventoryMovement {
	return &domain.InventoryMovement{
		ID:        fromUUID(row.ID),
		ProductID: fromUUID(row.ProductID),
		Type:      row.Type,
		Quantity:  row.Quantity,
		Reason:    fromText(row.Reason),
		CreatedAt: fromTimestamptz(row.CreatedAt),
		CreatedBy: fromNullUUID(row.CreatedBy),
		DeletedAt: fromNullTimestamptz(row.DeletedAt),
		DeletedBy: fromNullUUID(row.DeletedBy),
	}
}
