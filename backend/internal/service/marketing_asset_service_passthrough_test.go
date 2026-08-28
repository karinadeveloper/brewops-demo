package service

import (
	"context"
	"testing"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestMarketingAssetList_ReturnsActiveAssets(t *testing.T) {
	// Arrange
	assets := &mockMarketingAssetRepository{
		listFunc: func(_ context.Context) ([]domain.MarketingAsset, error) {
			return []domain.MarketingAsset{{Name: "Promo de verano"}}, nil
		},
	}
	svc := NewMarketingAssetService(assets, NewImageService(&mockStorageClient{}))

	// Act
	list, err := svc.List(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(list) != 1 || list[0].Name != "Promo de verano" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestMarketingAssetListTrash_ReturnsTrashedAssets(t *testing.T) {
	// Arrange
	email := "owner@brewops.mx"
	assets := &mockMarketingAssetRepository{
		listTrashFunc: func(_ context.Context) ([]domain.TrashedMarketingAsset, error) {
			return []domain.TrashedMarketingAsset{{MarketingAsset: domain.MarketingAsset{Name: "Descontinuado"}, DeletedByEmail: &email}}, nil
		},
	}
	svc := NewMarketingAssetService(assets, NewImageService(&mockStorageClient{}))

	// Act
	trashed, err := svc.ListTrash(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(trashed) != 1 || *trashed[0].DeletedByEmail != email {
		t.Fatalf("unexpected trashed assets: %+v", trashed)
	}
}
