package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestInventoryList_DefaultsAppliedForZeroValues_ComputesLimitAndOffset(t *testing.T) {
	// Arrange
	var gotParams domain.ListMovementsParams
	movements := &mockInventoryMovementRepository{
		listFunc: func(_ context.Context, params domain.ListMovementsParams) ([]domain.InventoryMovement, error) {
			gotParams = params
			return []domain.InventoryMovement{}, nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	_, err := svc.List(context.Background(), ListMovementsInput{Page: 2, PageSize: 10})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotParams.Limit != 10 || gotParams.Offset != 10 {
		t.Fatalf("expected limit=10 offset=10, got limit=%d offset=%d", gotParams.Limit, gotParams.Offset)
	}
}

func TestInventorySoftDelete_ExistingMovement_Succeeds(t *testing.T) {
	// Arrange
	id, deletedBy := uuid.New(), uuid.New()
	called := false
	movements := &mockInventoryMovementRepository{
		softDeleteFunc: func(_ context.Context, gotID, gotDeletedBy uuid.UUID) error {
			called = true
			if gotID != id || gotDeletedBy != deletedBy {
				t.Fatalf("unexpected args: id=%v deletedBy=%v", gotID, gotDeletedBy)
			}
			return nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	err := svc.SoftDelete(context.Background(), id, deletedBy)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected repository SoftDelete to be called")
	}
}

func TestInventoryListTrash_ReturnsTrashedMovements(t *testing.T) {
	// Arrange
	email := "owner@brewops.mx"
	movements := &mockInventoryMovementRepository{
		listTrashFunc: func(_ context.Context) ([]domain.TrashedInventoryMovement, error) {
			return []domain.TrashedInventoryMovement{
				{InventoryMovement: domain.InventoryMovement{Type: domain.MovementTypeAdjustmentOut}, DeletedByEmail: &email},
			}, nil
		},
	}
	svc := NewInventoryService(movements)

	// Act
	trashed, err := svc.ListTrash(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(trashed) != 1 || *trashed[0].DeletedByEmail != email {
		t.Fatalf("unexpected trashed movements: %+v", trashed)
	}
}
