package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestComputeTotalCents_VariousItems_SumsQuantityTimesUnitPrice(t *testing.T) {
	tests := []struct {
		name  string
		items []domain.SaleItemInput
		want  int64
	}{
		{"single item", []domain.SaleItemInput{{Quantity: 3, UnitPriceCents: 1500}}, 4500},
		{
			"multiple items",
			[]domain.SaleItemInput{
				{Quantity: 2, UnitPriceCents: 1000},
				{Quantity: 5, UnitPriceCents: 300},
			},
			3500,
		},
		{"no items", []domain.SaleItemInput{}, 0},
		{"zero unit price", []domain.SaleItemInput{{Quantity: 10, UnitPriceCents: 0}}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			got := computeTotalCents(tt.items)

			// Assert
			if got != tt.want {
				t.Fatalf("expected total %d, got %d", tt.want, got)
			}
		})
	}
}

func TestValidateSaleInput_VariousInputs_ReturnsExpectedError(t *testing.T) {
	validItems := []domain.SaleItemInput{{ProductID: uuid.New(), Quantity: 1, UnitPriceCents: 100}}

	tests := []struct {
		name          string
		items         []domain.SaleItemInput
		paymentMethod string
		wantErr       error
	}{
		{"valid input", validItems, "CASH", nil},
		{"valid transfer", validItems, "TRANSFER", nil},
		{"no items", []domain.SaleItemInput{}, "CASH", domain.ErrEmptySale},
		{"zero quantity", []domain.SaleItemInput{{ProductID: uuid.New(), Quantity: 0, UnitPriceCents: 100}}, "CASH", domain.ErrInvalidQuantity},
		{"negative quantity", []domain.SaleItemInput{{ProductID: uuid.New(), Quantity: -1, UnitPriceCents: 100}}, "CASH", domain.ErrInvalidQuantity},
		{"negative unit price", []domain.SaleItemInput{{ProductID: uuid.New(), Quantity: 1, UnitPriceCents: -1}}, "CASH", domain.ErrInvalidPrice},
		{"invalid payment method", validItems, "CREDIT", domain.ErrInvalidPaymentMethod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			err := validateSaleInput(tt.items, tt.paymentMethod)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreate_MultipleValidItems_ComputesTotalIgnoringAnyClientTotal(t *testing.T) {
	// Arrange
	productA, productB, createdBy := uuid.New(), uuid.New(), uuid.New()
	var gotSale domain.Sale
	sales := &mockSaleRepository{
		createFunc: func(_ context.Context, sale *domain.Sale, _ uuid.UUID) error {
			gotSale = *sale
			sale.ID = uuid.New()
			return nil
		},
	}
	svc := NewSaleService(sales)

	// Act — CreateSaleInput has no total_cents field at all, so there is no
	// client-submitted total for the service to even consider trusting.
	result, err := svc.Create(context.Background(), CreateSaleInput{
		Items: []domain.SaleItemInput{
			{ProductID: productA, Quantity: 2, UnitPriceCents: 4500},
			{ProductID: productB, Quantity: 1, UnitPriceCents: 2000},
		},
		PaymentMethod: "CASH",
		CreatedBy:     createdBy,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	const wantTotal = 2*4500 + 2000
	if gotSale.TotalCents != wantTotal {
		t.Fatalf("expected repository to receive total_cents %d, got %d", wantTotal, gotSale.TotalCents)
	}
	if result.TotalCents != wantTotal {
		t.Fatalf("expected returned sale total_cents %d, got %d", wantTotal, result.TotalCents)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
}

func TestCreate_ItemWithInsufficientStock_FailsCompletelyWithoutRepositorySideEffects(t *testing.T) {
	// Arrange
	callCount := 0
	sales := &mockSaleRepository{
		createFunc: func(_ context.Context, _ *domain.Sale, _ uuid.UUID) error {
			callCount++
			// Simulates SaleRepository.Create's real behavior: the whole DB
			// transaction rolled back, so it returns the error and mutates
			// nothing — no Sale, no SaleItem, no InventoryMovement, no
			// stock change persisted anywhere.
			return domain.ErrInsufficientStock
		},
	}
	svc := NewSaleService(sales)

	// Act
	_, err := svc.Create(context.Background(), CreateSaleInput{
		Items: []domain.SaleItemInput{
			{ProductID: uuid.New(), Quantity: 5, UnitPriceCents: 1000},
			{ProductID: uuid.New(), Quantity: 999, UnitPriceCents: 500},
		},
		PaymentMethod: "CASH",
		CreatedBy:     uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected repository Create to be called exactly once, got %d", callCount)
	}
}

func TestCreate_ConcurrentSaleChangedProductVersion_ReturnsErrOptimisticLockConflict(t *testing.T) {
	// Arrange
	sales := &mockSaleRepository{
		createFunc: func(_ context.Context, _ *domain.Sale, _ uuid.UUID) error {
			return domain.ErrOptimisticLockConflict
		},
	}
	svc := NewSaleService(sales)

	// Act
	_, err := svc.Create(context.Background(), CreateSaleInput{
		Items:         []domain.SaleItemInput{{ProductID: uuid.New(), Quantity: 1, UnitPriceCents: 1000}},
		PaymentMethod: "CASH",
		CreatedBy:     uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrOptimisticLockConflict) {
		t.Fatalf("expected ErrOptimisticLockConflict, got %v", err)
	}
}

func TestCreate_InvalidInput_ReturnsValidationErrorWithoutCallingRepository(t *testing.T) {
	// Arrange
	sales := &mockSaleRepository{
		createFunc: func(_ context.Context, _ *domain.Sale, _ uuid.UUID) error {
			t.Fatal("repository Create should not be called when input validation fails")
			return nil
		},
	}
	svc := NewSaleService(sales)

	// Act
	_, err := svc.Create(context.Background(), CreateSaleInput{
		Items:         []domain.SaleItemInput{},
		PaymentMethod: "CASH",
		CreatedBy:     uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrEmptySale) {
		t.Fatalf("expected ErrEmptySale, got %v", err)
	}
}

func TestSaleRestore_ExistingSale_DoesNotReapplyStockEffect(t *testing.T) {
	// Arrange
	id := uuid.New()
	restored := domain.Sale{ID: id, TotalCents: 5000}
	sales := &mockSaleRepository{
		restoreFunc: func(_ context.Context, gotID uuid.UUID) (*domain.Sale, error) {
			if gotID != id {
				t.Fatalf("expected restore for %v, got %v", id, gotID)
			}
			return &restored, nil
		},
	}
	svc := NewSaleService(sales)

	// Act
	result, err := svc.Restore(context.Background(), id)

	// Assert — Restore only delegates to the repository's Restore (which
	// clears deleted_at/deleted_by); nothing here touches current_stock or
	// recreates movements.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != id {
		t.Fatalf("expected restored sale %v, got %v", id, result.ID)
	}
}
