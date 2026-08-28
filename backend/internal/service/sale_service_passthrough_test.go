package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestSaleGet_ExistingSale_ReturnsSale(t *testing.T) {
	// Arrange
	id := uuid.New()
	sales := &mockSaleRepository{
		getByIDFunc: func(_ context.Context, gotID uuid.UUID) (*domain.Sale, error) {
			if gotID != id {
				t.Fatalf("expected lookup for %v, got %v", id, gotID)
			}
			return &domain.Sale{ID: id, TotalCents: 4500}, nil
		},
	}
	svc := NewSaleService(sales)

	// Act
	sale, err := svc.Get(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sale.ID != id {
		t.Fatalf("expected sale %v, got %v", id, sale.ID)
	}
}

func TestSaleList_DefaultsAppliedForZeroValues_ComputesLimitAndOffset(t *testing.T) {
	// Arrange
	var gotParams domain.ListSalesParams
	sales := &mockSaleRepository{
		listFunc: func(_ context.Context, params domain.ListSalesParams) ([]domain.Sale, error) {
			gotParams = params
			return []domain.Sale{}, nil
		},
	}
	svc := NewSaleService(sales)

	// Act
	_, err := svc.List(context.Background(), ListSalesInput{Page: 3, PageSize: 5})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotParams.Limit != 5 || gotParams.Offset != 10 {
		t.Fatalf("expected limit=5 offset=10, got limit=%d offset=%d", gotParams.Limit, gotParams.Offset)
	}
}

func TestSaleSoftDelete_ExistingSale_Succeeds(t *testing.T) {
	// Arrange
	id, deletedBy := uuid.New(), uuid.New()
	called := false
	sales := &mockSaleRepository{
		softDeleteFunc: func(_ context.Context, gotID, gotDeletedBy uuid.UUID) error {
			called = true
			if gotID != id || gotDeletedBy != deletedBy {
				t.Fatalf("unexpected args: id=%v deletedBy=%v", gotID, gotDeletedBy)
			}
			return nil
		},
	}
	svc := NewSaleService(sales)

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

func TestSaleListTrash_ReturnsTrashedSales(t *testing.T) {
	// Arrange
	email := "owner@brewops.mx"
	sales := &mockSaleRepository{
		listTrashFunc: func(_ context.Context) ([]domain.TrashedSale, error) {
			return []domain.TrashedSale{{Sale: domain.Sale{TotalCents: 1000}, DeletedByEmail: &email}}, nil
		},
	}
	svc := NewSaleService(sales)

	// Act
	trashed, err := svc.ListTrash(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(trashed) != 1 || *trashed[0].DeletedByEmail != email {
		t.Fatalf("unexpected trashed sales: %+v", trashed)
	}
}
