package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestValidatePricing_VariousInputs_ReturnsExpectedError(t *testing.T) {
	tests := []struct {
		name           string
		salePriceCents int64
		costCents      int64
		currentStock   int32
		minStock       int32
		wantErr        error
	}{
		{"all valid", 1000, 500, 10, 2, nil},
		{"zero values are valid", 0, 0, 0, 0, nil},
		{"negative sale price", -1, 500, 10, 2, domain.ErrInvalidPrice},
		{"negative cost", 1000, -1, 10, 2, domain.ErrInvalidPrice},
		{"negative current stock", 1000, 500, -1, 2, domain.ErrInvalidStock},
		{"negative min stock", 1000, 500, 10, -1, domain.ErrInvalidStock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			err := validatePricing(tt.salePriceCents, tt.costCents, tt.currentStock, tt.minStock)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreate_ValidInput_ReturnsProduct(t *testing.T) {
	// Arrange
	var created domain.Product
	products := &mockProductRepository{
		createFunc: func(_ context.Context, p *domain.Product) error {
			p.ID = uuid.New()
			created = *p
			return nil
		},
	}
	svc := NewProductService(products)

	// Act
	result, err := svc.Create(context.Background(), CreateProductInput{
		Name:           "Jugo de naranja 1L",
		Category:       "juice",
		SalePriceCents: 4500,
		CostCents:      2000,
		CurrentStock:   50,
		MinStock:       10,
		CreatedBy:      uuid.New(),
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != created.ID {
		t.Fatalf("expected created product to be returned, got %v", result)
	}
}

func TestCreate_NegativePrice_ReturnsErrInvalidPrice(t *testing.T) {
	// Arrange
	products := &mockProductRepository{
		createFunc: func(_ context.Context, _ *domain.Product) error {
			t.Fatal("repository Create should not be called when validation fails")
			return nil
		},
	}
	svc := NewProductService(products)

	// Act
	_, err := svc.Create(context.Background(), CreateProductInput{
		Name:           "Jugo de naranja 1L",
		Category:       "juice",
		SalePriceCents: -100,
		CostCents:      2000,
		CurrentStock:   50,
		MinStock:       10,
		CreatedBy:      uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrInvalidPrice) {
		t.Fatalf("expected ErrInvalidPrice, got %v", err)
	}
}

func TestRestore_PositiveStock_Succeeds(t *testing.T) {
	// Arrange
	id := uuid.New()
	restored := domain.Product{ID: id, CurrentStock: 5}
	products := &mockProductRepository{
		getTrashedByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
			return &domain.Product{ID: id, CurrentStock: 5}, nil
		},
		restoreFunc: func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
			return &restored, nil
		},
	}
	svc := NewProductService(products)

	// Act
	result, err := svc.Restore(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != id {
		t.Fatalf("expected restored product with id %v, got %v", id, result.ID)
	}
}

func TestRestore_NegativeStock_ReturnsErrInvalidRestoreState(t *testing.T) {
	// Arrange
	id := uuid.New()
	products := &mockProductRepository{
		getTrashedByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
			return &domain.Product{ID: id, CurrentStock: -5}, nil
		},
		restoreFunc: func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
			t.Fatal("repository Restore should not be called when stock is negative")
			return nil, nil
		},
	}
	svc := NewProductService(products)

	// Act
	_, err := svc.Restore(context.Background(), id)

	// Assert
	if !errors.Is(err, domain.ErrInvalidRestoreState) {
		t.Fatalf("expected ErrInvalidRestoreState, got %v", err)
	}
}

func TestUpdate_StaleVersion_ReturnsErrOptimisticLockConflict(t *testing.T) {
	// Arrange
	id := uuid.New()
	products := &mockProductRepository{
		getByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.Product, error) {
			return &domain.Product{ID: id, Version: 2}, nil
		},
		updateFunc: func(_ context.Context, _ *domain.Product) (*domain.Product, error) {
			return nil, domain.ErrOptimisticLockConflict
		},
	}
	svc := NewProductService(products)

	// Act
	_, err := svc.Update(context.Background(), UpdateProductInput{
		ID:             id,
		Version:        1, // stale — the row is already at version 2
		Name:           "Jugo de naranja 1L",
		Category:       "juice",
		SalePriceCents: 4500,
		CostCents:      2000,
		CurrentStock:   50,
		MinStock:       10,
		UpdatedBy:      uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrOptimisticLockConflict) {
		t.Fatalf("expected ErrOptimisticLockConflict, got %v", err)
	}
}
