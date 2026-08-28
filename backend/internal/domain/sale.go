package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrSaleNotFound is returned when no sale matches the lookup in the
	// scope being queried (active or trashed).
	ErrSaleNotFound = errors.New("sale not found")
	// ErrEmptySale is returned when a sale is submitted with no items.
	ErrEmptySale = errors.New("sale must include at least one item")
	// ErrInvalidPaymentMethod is returned for any payment method outside
	// CASH/TRANSFER.
	ErrInvalidPaymentMethod = errors.New("payment method must be CASH or TRANSFER")
)

// SaleItem mirrors the sale_items table — one line per product sold in a
// Sale, at the unit price recorded at the time of sale.
type SaleItem struct {
	ID             uuid.UUID
	SaleID         uuid.UUID
	ProductID      uuid.UUID
	Quantity       int32
	UnitPriceCents int64
	CreatedAt      time.Time
}

// Sale mirrors the sales table. TotalCents is always computed server-side
// from Items — never trusted from client input.
type Sale struct {
	ID            uuid.UUID
	Items         []SaleItem
	TotalCents    int64
	PaymentMethod string
	CreatedAt     time.Time
	CreatedBy     *uuid.UUID
	DeletedAt     *time.Time
	DeletedBy     *uuid.UUID
}

// TrashedSale is a soft-deleted Sale enriched with the email of the user who
// deleted it, for the trash view.
type TrashedSale struct {
	Sale
	DeletedByEmail *string
}

// SaleItemInput is one requested line of a sale before persistence: the
// product, how many units, and the unit price to record.
type SaleItemInput struct {
	ProductID      uuid.UUID
	Quantity       int32
	UnitPriceCents int64
}

// ListSalesParams filters and paginates the active sales history.
type ListSalesParams struct {
	From   *time.Time
	To     *time.Time
	Limit  int32
	Offset int32
}

// SaleRepository is the persistence boundary the sale service depends on.
type SaleRepository interface {
	// Create performs the entire sale as a single DB transaction: per-item
	// stock verification (product exists, isn't soft-deleted, has enough
	// current_stock), the Sale and SaleItem inserts, one auto-generated SALE
	// InventoryMovement per item, and an optimistic-concurrency-guarded
	// stock decrement per product. If any item fails verification, or a
	// product's version changed between verification and the decrement,
	// nothing is persisted — the whole transaction rolls back. sale.Items,
	// sale.TotalCents, and sale.PaymentMethod must be populated by the
	// caller before calling Create; sale.ID/CreatedAt/CreatedBy are filled
	// in on success.
	Create(ctx context.Context, sale *Sale, createdBy uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Sale, error)
	List(ctx context.Context, params ListSalesParams) ([]Sale, error)
	SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error
	// Restore clears deleted_at/deleted_by only — it never reverses the
	// original stock decrement or recreates SALE movements. See
	// service.SaleService.Restore for why.
	Restore(ctx context.Context, id uuid.UUID) (*Sale, error)
	ListTrash(ctx context.Context) ([]TrashedSale, error)
}
