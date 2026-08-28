package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Movement types — the only valid values for InventoryMovement.Type. SALE is
// never accepted from POST /inventory/movements; it is generated exclusively
// by the sales flow (see SaleRepository.Create).
const (
	MovementTypePurchase      = "PURCHASE"
	MovementTypeSale          = "SALE"
	MovementTypeAdjustmentIn  = "ADJUSTMENT_IN"
	MovementTypeAdjustmentOut = "ADJUSTMENT_OUT"
)

var (
	// ErrMovementNotFound is returned when no inventory movement matches the
	// lookup in the scope being queried (active or trashed).
	ErrMovementNotFound = errors.New("inventory movement not found")
	// ErrManualSaleMovement is returned when a caller tries to create a
	// manual SALE movement via POST /inventory/movements. SALE movements are
	// only ever generated internally by POST /sales.
	ErrManualSaleMovement = errors.New("SALE movements cannot be created manually; they are generated automatically by POST /sales")
	// ErrInvalidMovementType is returned for any type outside the known
	// movement types.
	ErrInvalidMovementType = errors.New("invalid movement type")
	// ErrInvalidQuantity is returned when quantity is not a positive
	// magnitude, mirroring the DB's CHECK (quantity > 0).
	ErrInvalidQuantity = errors.New("quantity must be greater than zero")
)

// InventoryMovement mirrors the inventory_movements table. Quantity is
// always a positive magnitude — direction is derived exclusively from Type,
// never from Reason (free text for human context only).
type InventoryMovement struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	Type      string
	Quantity  int32
	Reason    *string
	CreatedAt time.Time
	CreatedBy *uuid.UUID
	DeletedAt *time.Time
	DeletedBy *uuid.UUID
}

// TrashedInventoryMovement is a soft-deleted InventoryMovement enriched with
// the email of the user who deleted it, for the trash view.
type TrashedInventoryMovement struct {
	InventoryMovement
	DeletedByEmail *string
}

// ListMovementsParams filters and paginates the active movement history.
type ListMovementsParams struct {
	ProductID *uuid.UUID
	From      *time.Time
	To        *time.Time
	Limit     int32
	Offset    int32
}

// InventoryMovementRepository is the persistence boundary the inventory
// service depends on.
type InventoryMovementRepository interface {
	// CreateManual inserts m and applies stockDelta to the product's
	// current_stock in a single DB transaction — never as two separate
	// writes. stockDelta is signed (positive for PURCHASE/ADJUSTMENT_IN,
	// negative for ADJUSTMENT_OUT); the repository has no opinion on
	// movement type, only on applying the delta atomically. Returns
	// ErrProductNotFound if the product doesn't exist or is soft-deleted,
	// or ErrInsufficientStock if applying the delta would take current_stock
	// below zero.
	CreateManual(ctx context.Context, m *InventoryMovement, stockDelta int32) error
	GetByID(ctx context.Context, id uuid.UUID) (*InventoryMovement, error)
	List(ctx context.Context, params ListMovementsParams) ([]InventoryMovement, error)
	SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error
	GetTrashedByID(ctx context.Context, id uuid.UUID) (*InventoryMovement, error)
	// Restore clears deleted_at/deleted_by only — it never re-applies the
	// movement's original effect on current_stock. See
	// service.InventoryService.Restore for why.
	Restore(ctx context.Context, id uuid.UUID) (*InventoryMovement, error)
	ListTrash(ctx context.Context) ([]TrashedInventoryMovement, error)
}
