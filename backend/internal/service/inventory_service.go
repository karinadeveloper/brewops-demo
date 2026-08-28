package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// InventoryService implements manual inventory movement creation and the
// soft-delete/trash/restore business rules on top of a
// domain.InventoryMovementRepository.
type InventoryService struct {
	movements domain.InventoryMovementRepository
}

func NewInventoryService(movements domain.InventoryMovementRepository) *InventoryService {
	return &InventoryService{movements: movements}
}

// manualMovementDelta computes the signed current_stock change for a manual
// inventory movement, and rejects anything that isn't a valid manual type.
// SALE is deliberately excluded here — it is never created through this
// path, only via the sales flow. Pure and exhaustively tested per
// CLAUDE.md's 100%-coverage rule for stock calculation logic.
func manualMovementDelta(movementType string, quantity int32) (int32, error) {
	if quantity <= 0 {
		return 0, domain.ErrInvalidQuantity
	}
	switch movementType {
	case domain.MovementTypePurchase, domain.MovementTypeAdjustmentIn:
		return quantity, nil
	case domain.MovementTypeAdjustmentOut:
		return -quantity, nil
	case domain.MovementTypeSale:
		return 0, domain.ErrManualSaleMovement
	default:
		return 0, domain.ErrInvalidMovementType
	}
}

type CreateMovementInput struct {
	ProductID uuid.UUID
	Type      string
	Quantity  int32
	Reason    *string
	CreatedBy uuid.UUID
}

// CreateManual validates the movement type/quantity, then delegates the
// atomic insert-and-stock-update to the repository. The delta's sign
// (increase for PURCHASE/ADJUSTMENT_IN, decrease for ADJUSTMENT_OUT) is
// resolved here so it can be unit tested without a database.
func (s *InventoryService) CreateManual(ctx context.Context, in CreateMovementInput) (*domain.InventoryMovement, error) {
	delta, err := manualMovementDelta(in.Type, in.Quantity)
	if err != nil {
		return nil, err
	}

	m := &domain.InventoryMovement{
		ProductID: in.ProductID,
		Type:      in.Type,
		Quantity:  in.Quantity,
		Reason:    in.Reason,
		CreatedBy: &in.CreatedBy,
	}
	if err := s.movements.CreateManual(ctx, m, delta); err != nil {
		return nil, fmt.Errorf("inventory movement: create: %w", err)
	}
	return m, nil
}

type ListMovementsInput struct {
	ProductID *uuid.UUID
	From      *time.Time
	To        *time.Time
	Page      int32
	PageSize  int32
}

func (s *InventoryService) List(ctx context.Context, in ListMovementsInput) ([]domain.InventoryMovement, error) {
	limit := in.PageSize
	if limit <= 0 {
		limit = defaultPageSize
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	return s.movements.List(ctx, domain.ListMovementsParams{
		ProductID: in.ProductID,
		From:      in.From,
		To:        in.To,
		Limit:     limit,
		Offset:    offset,
	})
}

func (s *InventoryService) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return s.movements.SoftDelete(ctx, id, deletedBy)
}

func (s *InventoryService) ListTrash(ctx context.Context) ([]domain.TrashedInventoryMovement, error) {
	return s.movements.ListTrash(ctx)
}

// Restore brings a soft-deleted movement back into the active trash-free
// view. Unlike ProductService.Restore, it never re-applies any effect on
// current_stock — see domain.InventoryMovementRepository.Restore for why: a
// movement's stock effect already happened atomically when it was created,
// so "restoring" it is purely an audit-trail visibility change, not an
// operation to re-run. There is accordingly no invariant left to
// revalidate here, unlike a product restore which must re-check
// current_stock >= 0.
func (s *InventoryService) Restore(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	m, err := s.movements.Restore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("inventory movement: restore: %w", err)
	}
	return m, nil
}
