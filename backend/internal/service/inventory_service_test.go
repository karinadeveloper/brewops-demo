package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestManualMovementDelta_VariousInputs_ReturnsExpectedDeltaOrError(t *testing.T) {
	tests := []struct {
		name         string
		movementType string
		quantity     int32
		wantDelta    int32
		wantErr      error
	}{
		{"purchase increases stock", domain.MovementTypePurchase, 10, 10, nil},
		{"adjustment in increases stock", domain.MovementTypeAdjustmentIn, 5, 5, nil},
		{"adjustment out decreases stock", domain.MovementTypeAdjustmentOut, 5, -5, nil},
		{"manual sale is rejected", domain.MovementTypeSale, 5, 0, domain.ErrManualSaleMovement},
		{"unknown type is rejected", "GIFT", 5, 0, domain.ErrInvalidMovementType},
		{"zero quantity is rejected", domain.MovementTypePurchase, 0, 0, domain.ErrInvalidQuantity},
		{"negative quantity is rejected", domain.MovementTypeAdjustmentOut, -1, 0, domain.ErrInvalidQuantity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			delta, err := manualMovementDelta(tt.movementType, tt.quantity)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if err == nil && delta != tt.wantDelta {
				t.Fatalf("expected delta %d, got %d", tt.wantDelta, delta)
			}
		})
	}
}

func TestCreateManual_ValidPurchase_ReturnsMovement(t *testing.T) {
	// Arrange
	productID, createdBy := uuid.New(), uuid.New()
	var gotDelta int32
	movements := &mockInventoryMovementRepository{
		createManualFunc: func(_ context.Context, m *domain.InventoryMovement, stockDelta int32) error {
			gotDelta = stockDelta
			m.ID = uuid.New()
			return nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	movement, err := svc.CreateManual(context.Background(), CreateMovementInput{
		ProductID: productID,
		Type:      domain.MovementTypePurchase,
		Quantity:  20,
		CreatedBy: createdBy,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotDelta != 20 {
		t.Fatalf("expected repository to receive delta 20, got %d", gotDelta)
	}
	if movement.ProductID != productID {
		t.Fatalf("expected movement for product %v, got %v", productID, movement.ProductID)
	}
}

func TestCreateManual_AdjustmentOutWouldGoNegative_ReturnsErrInsufficientStock(t *testing.T) {
	// Arrange
	movements := &mockInventoryMovementRepository{
		createManualFunc: func(_ context.Context, _ *domain.InventoryMovement, _ int32) error {
			return domain.ErrInsufficientStock
		},
	}
	svc := NewInventoryService(movements)

	// Act
	_, err := svc.CreateManual(context.Background(), CreateMovementInput{
		ProductID: uuid.New(),
		Type:      domain.MovementTypeAdjustmentOut,
		Quantity:  100,
		CreatedBy: uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestCreateManual_ProductSoftDeleted_ReturnsErrProductNotFound(t *testing.T) {
	// Arrange
	movements := &mockInventoryMovementRepository{
		createManualFunc: func(_ context.Context, _ *domain.InventoryMovement, _ int32) error {
			return domain.ErrProductNotFound
		},
	}
	svc := NewInventoryService(movements)

	// Act
	_, err := svc.CreateManual(context.Background(), CreateMovementInput{
		ProductID: uuid.New(),
		Type:      domain.MovementTypeAdjustmentIn,
		Quantity:  5,
		CreatedBy: uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

func TestCreateManual_ManualSaleType_ReturnsErrManualSaleMovementWithoutCallingRepository(t *testing.T) {
	// Arrange
	movements := &mockInventoryMovementRepository{
		createManualFunc: func(_ context.Context, _ *domain.InventoryMovement, _ int32) error {
			t.Fatal("repository CreateManual should not be called for a manual SALE movement")
			return nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	_, err := svc.CreateManual(context.Background(), CreateMovementInput{
		ProductID: uuid.New(),
		Type:      domain.MovementTypeSale,
		Quantity:  5,
		CreatedBy: uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrManualSaleMovement) {
		t.Fatalf("expected ErrManualSaleMovement, got %v", err)
	}
}

func TestInventoryRestore_ExistingMovement_DoesNotReapplyStockEffect(t *testing.T) {
	// Arrange
	id := uuid.New()
	restored := domain.InventoryMovement{ID: id, Type: domain.MovementTypeAdjustmentOut, Quantity: 50}
	movements := &mockInventoryMovementRepository{
		restoreFunc: func(_ context.Context, gotID uuid.UUID) (*domain.InventoryMovement, error) {
			if gotID != id {
				t.Fatalf("expected restore for %v, got %v", id, gotID)
			}
			return &restored, nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	result, err := svc.Restore(context.Background(), id)

	// Assert — Restore only delegates to the repository's Restore (which
	// clears deleted_at/deleted_by); nothing here touches current_stock.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != id {
		t.Fatalf("expected restored movement %v, got %v", id, result.ID)
	}
}
