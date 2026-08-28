package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestNormalizeSuggestedMovementType_VariousTypes_ReturnsExpected(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"purchase is kept", domain.MovementTypePurchase, domain.MovementTypePurchase},
		{"adjustment in is kept", domain.MovementTypeAdjustmentIn, domain.MovementTypeAdjustmentIn},
		{"adjustment out is kept", domain.MovementTypeAdjustmentOut, domain.MovementTypeAdjustmentOut},
		{"sale is discarded", domain.MovementTypeSale, ""},
		{"unknown value is discarded", "GIFT", ""},
		{"empty stays empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			got := normalizeSuggestedMovementType(tt.input)

			// Assert
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestBestProductMatch_VariousInputs_ReturnsExpectedMatchOrNil(t *testing.T) {
	products := []domain.Product{
		{Name: "Jugo de naranja 1L"},
		{Name: "Jugo de mango 1L"},
		{Name: "Agua mineral 600ml"},
	}

	tests := []struct {
		name      string
		mentioned string
		wantName  string // "" means expect nil
	}{
		{"substring match against a longer product name", "naranjas", "Jugo de naranja 1L"},
		{"substring match, different product", "mango", "Jugo de mango 1L"},
		{"exact match, case-insensitive", "AGUA MINERAL 600ML", "Agua mineral 600ml"},
		{"unrelated word has no reasonable match", "refrescos de cola", ""},
		{"empty mention has no match", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			match := bestProductMatch(tt.mentioned, products)

			// Assert
			if tt.wantName == "" {
				if match != nil {
					t.Fatalf("expected no match, got %q", match.Name)
				}
				return
			}
			if match == nil || match.Name != tt.wantName {
				t.Fatalf("expected match %q, got %v", tt.wantName, match)
			}
		})
	}
}

func TestSuggest_ValidResponse_ReturnsMatchedSuggestions(t *testing.T) {
	// Arrange
	orangeID := uuid.New()
	ai := &mockInventorySuggestionClient{
		suggestFunc: func(_ context.Context, text string) (string, error) {
			return `{"movements":[{"product_name":"naranjas","type":"PURCHASE","quantity":50,"unit_cost_cents":500}]}`, nil
		},
	}
	products := &mockProductRepository{
		listFunc: func(_ context.Context, _ domain.ListProductsParams) ([]domain.Product, error) {
			return []domain.Product{{ID: orangeID, Name: "Jugo de naranja 1L"}}, nil
		},
	}
	svc := NewInventorySuggestionService(ai, products)

	// Act
	suggestions, err := svc.Suggest(context.Background(), "compré 50 naranjas a 5 pesos cada uno")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	s := suggestions[0]
	if s.ProductMentioned != "naranjas" {
		t.Fatalf("expected ProductMentioned 'naranjas', got %q", s.ProductMentioned)
	}
	if s.ProductMatch == nil || *s.ProductMatch != orangeID {
		t.Fatalf("expected ProductMatch %s, got %v", orangeID, s.ProductMatch)
	}
	if s.Type != domain.MovementTypePurchase || s.Quantity != 50 || s.UnitCostCents != 500 {
		t.Fatalf("unexpected suggestion fields: %+v", s)
	}
}

func TestSuggest_ProductWithoutReasonableMatch_ReturnsNilProductMatch(t *testing.T) {
	// Arrange
	ai := &mockInventorySuggestionClient{
		suggestFunc: func(_ context.Context, _ string) (string, error) {
			return `{"movements":[{"product_name":"tamarindo deshidratado","type":"PURCHASE","quantity":10,"unit_cost_cents":300}]}`, nil
		},
	}
	products := &mockProductRepository{
		listFunc: func(_ context.Context, _ domain.ListProductsParams) ([]domain.Product, error) {
			return []domain.Product{{Name: "Jugo de naranja 1L"}, {Name: "Agua mineral 600ml"}}, nil
		},
	}
	svc := NewInventorySuggestionService(ai, products)

	// Act
	suggestions, err := svc.Suggest(context.Background(), "compré 10 tamarindo deshidratado")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(suggestions) != 1 || suggestions[0].ProductMatch != nil {
		t.Fatalf("expected a single suggestion with a nil ProductMatch, got %+v", suggestions)
	}
}

func TestSuggest_AIClientFails_ReturnsErrAISuggestionFailed(t *testing.T) {
	// Arrange
	ai := &mockInventorySuggestionClient{
		suggestFunc: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("context deadline exceeded")
		},
	}
	products := &mockProductRepository{
		listFunc: func(_ context.Context, _ domain.ListProductsParams) ([]domain.Product, error) {
			t.Fatal("products should not be listed when the AI call itself fails")
			return nil, nil
		},
	}
	svc := NewInventorySuggestionService(ai, products)

	// Act
	_, err := svc.Suggest(context.Background(), "compré 50 naranjas")

	// Assert
	if !errors.Is(err, domain.ErrAISuggestionFailed) {
		t.Fatalf("expected ErrAISuggestionFailed, got %v", err)
	}
}

func TestSuggest_MalformedAIResponse_ReturnsErrAISuggestionFailed(t *testing.T) {
	// Arrange
	ai := &mockInventorySuggestionClient{
		suggestFunc: func(_ context.Context, _ string) (string, error) {
			return "not valid json at all", nil
		},
	}
	products := &mockProductRepository{
		listFunc: func(_ context.Context, _ domain.ListProductsParams) ([]domain.Product, error) {
			t.Fatal("products should not be listed when the AI response can't be parsed")
			return nil, nil
		},
	}
	svc := NewInventorySuggestionService(ai, products)

	// Act
	_, err := svc.Suggest(context.Background(), "compré 50 naranjas")

	// Assert
	if !errors.Is(err, domain.ErrAISuggestionFailed) {
		t.Fatalf("expected ErrAISuggestionFailed, got %v", err)
	}
}
