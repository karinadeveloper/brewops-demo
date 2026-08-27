package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrProductNotFound is returned when no product matches the lookup in
	// the scope being queried (active or trashed).
	ErrProductNotFound = errors.New("product not found")
	// ErrInsufficientStock is returned when an operation would take
	// current_stock below zero.
	ErrInsufficientStock = errors.New("insufficient stock")
	// ErrOptimisticLockConflict is returned when a PATCH's submitted
	// version no longer matches the row's current version.
	ErrOptimisticLockConflict = errors.New("product was modified by another request, refresh and try again")
	// ErrInvalidRestoreState is returned when restoring a product would
	// leave it in a state that violates a business invariant (e.g.
	// negative stock). Wrapped with details via fmt.Errorf("%w: ...").
	ErrInvalidRestoreState = errors.New("cannot restore")
	// ErrInvalidPrice is returned when sale_price_cents or cost_cents is negative.
	ErrInvalidPrice = errors.New("price and cost must not be negative")
	// ErrInvalidStock is returned when current_stock or min_stock is negative.
	ErrInvalidStock = errors.New("stock and min stock must not be negative")
)

// Product mirrors the products table, including the version column used
// for optimistic concurrency and the soft-delete/audit columns.
type Product struct {
	ID             uuid.UUID
	Name           string
	Category       string
	SalePriceCents int64
	CostCents      int64
	CurrentStock   int32
	MinStock       int32
	ImageURL       *string
	Version        int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdatedBy      *uuid.UUID
	DeletedAt      *time.Time
	DeletedBy      *uuid.UUID
}

// TrashedProduct is a soft-deleted Product enriched with the email of the
// user who deleted it, for the trash view.
type TrashedProduct struct {
	Product
	DeletedByEmail *string
}

// ListProductsParams filters and paginates the active product catalog.
type ListProductsParams struct {
	Category *string
	Limit    int32
	Offset   int32
}

// ProductRepository is the persistence boundary the product service depends
// on: CRUD, plus the soft-delete/trash/restore and low-stock query the
// business rules require.
type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	List(ctx context.Context, params ListProductsParams) ([]Product, error)
	Update(ctx context.Context, p *Product) (*Product, error)
	SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error
	// GetTrashedByID fetches a soft-deleted product by ID — used by Restore
	// to re-check current_stock before clearing deleted_at/deleted_by.
	GetTrashedByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Restore(ctx context.Context, id uuid.UUID) (*Product, error)
	ListTrash(ctx context.Context) ([]TrashedProduct, error)
	ListLowStock(ctx context.Context) ([]Product, error)
}
