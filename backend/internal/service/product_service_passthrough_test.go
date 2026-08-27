package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestGet_ExistingProduct_ReturnsProduct(t *testing.T) {
	// Arrange
	id := uuid.New()
	products := &mockProductRepository{
		getByIDFunc: func(_ context.Context, gotID uuid.UUID) (*domain.Product, error) {
			if gotID != id {
				t.Fatalf("expected lookup for %v, got %v", id, gotID)
			}
			return &domain.Product{ID: id, Name: "Agua 600ml"}, nil
		},
	}
	svc := NewProductService(products)

	// Act
	product, err := svc.Get(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if product.ID != id {
		t.Fatalf("expected product %v, got %v", id, product.ID)
	}
}

func TestList_DefaultsAppliedForZeroValues_ComputesLimitAndOffset(t *testing.T) {
	// Arrange
	var gotParams domain.ListProductsParams
	products := &mockProductRepository{
		listFunc: func(_ context.Context, params domain.ListProductsParams) ([]domain.Product, error) {
			gotParams = params
			return []domain.Product{}, nil
		},
	}
	svc := NewProductService(products)

	// Act
	_, err := svc.List(context.Background(), ListProductsInput{Page: 2, PageSize: 10})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotParams.Limit != 10 || gotParams.Offset != 10 {
		t.Fatalf("expected limit=10 offset=10, got limit=%d offset=%d", gotParams.Limit, gotParams.Offset)
	}
}

func TestSoftDelete_ExistingProduct_Succeeds(t *testing.T) {
	// Arrange
	id, deletedBy := uuid.New(), uuid.New()
	called := false
	products := &mockProductRepository{
		softDeleteFunc: func(_ context.Context, gotID, gotDeletedBy uuid.UUID) error {
			called = true
			if gotID != id || gotDeletedBy != deletedBy {
				t.Fatalf("unexpected args: id=%v deletedBy=%v", gotID, gotDeletedBy)
			}
			return nil
		},
	}
	svc := NewProductService(products)

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

func TestListTrash_ReturnsTrashedProducts(t *testing.T) {
	// Arrange
	email := "owner@brewops.mx"
	products := &mockProductRepository{
		listTrashFunc: func(_ context.Context) ([]domain.TrashedProduct, error) {
			return []domain.TrashedProduct{{Product: domain.Product{Name: "Descontinuado"}, DeletedByEmail: &email}}, nil
		},
	}
	svc := NewProductService(products)

	// Act
	trashed, err := svc.ListTrash(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(trashed) != 1 || *trashed[0].DeletedByEmail != email {
		t.Fatalf("unexpected trashed products: %+v", trashed)
	}
}

func TestListLowStock_ReturnsProductsAtOrBelowMinStock(t *testing.T) {
	// Arrange
	products := &mockProductRepository{
		listLowStockFunc: func(_ context.Context) ([]domain.Product, error) {
			return []domain.Product{{Name: "Jugo de mango 1L", CurrentStock: 1, MinStock: 5}}, nil
		},
	}
	svc := NewProductService(products)

	// Act
	lowStock, err := svc.ListLowStock(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(lowStock) != 1 {
		t.Fatalf("expected 1 low-stock product, got %d", len(lowStock))
	}
}
