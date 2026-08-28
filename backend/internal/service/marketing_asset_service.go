package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// MarketingAssetService implements marketing asset creation (via the
// shared ImageService upload path) and the soft-delete/trash/restore
// business rules on top of a domain.MarketingAssetRepository.
type MarketingAssetService struct {
	assets domain.MarketingAssetRepository
	images *ImageService
}

func NewMarketingAssetService(assets domain.MarketingAssetRepository, images *ImageService) *MarketingAssetService {
	return &MarketingAssetService{assets: assets, images: images}
}

// validateMarketingAssetType is pure and exhaustively tested.
func validateMarketingAssetType(assetType string) error {
	if assetType != domain.MarketingAssetTypeProduct && assetType != domain.MarketingAssetTypePromotion {
		return domain.ErrInvalidMarketingAssetType
	}
	return nil
}

type CreateMarketingAssetInput struct {
	Name        string
	Type        string
	ImageData   []byte
	ContentType string
	CreatedBy   uuid.UUID
}

func (s *MarketingAssetService) Create(ctx context.Context, in CreateMarketingAssetInput) (*domain.MarketingAsset, error) {
	if err := validateMarketingAssetType(in.Type); err != nil {
		return nil, err
	}

	url, err := s.images.Upload(ctx, in.ImageData, in.ContentType)
	if err != nil {
		return nil, err
	}

	asset := &domain.MarketingAsset{
		Name:      in.Name,
		ImageURL:  url,
		Type:      in.Type,
		CreatedBy: &in.CreatedBy,
	}
	if err := s.assets.Create(ctx, asset); err != nil {
		return nil, fmt.Errorf("marketing asset: create: %w", err)
	}
	return asset, nil
}

func (s *MarketingAssetService) List(ctx context.Context) ([]domain.MarketingAsset, error) {
	return s.assets.List(ctx)
}

// SoftDelete removes the asset from the active list only — it deliberately
// never deletes the underlying file from StorageClient. Unlike a typical
// "delete", the file must survive so Restore can bring the record back
// without re-uploading the image. See ListTrash/Restore for the rest of
// this lifecycle.
func (s *MarketingAssetService) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return s.assets.SoftDelete(ctx, id, deletedBy)
}

func (s *MarketingAssetService) ListTrash(ctx context.Context) ([]domain.TrashedMarketingAsset, error) {
	return s.assets.ListTrash(ctx)
}

// Restore reactivates the record; ImageURL still points at the same file
// that was never deleted, so no re-upload is needed.
func (s *MarketingAssetService) Restore(ctx context.Context, id uuid.UUID) (*domain.MarketingAsset, error) {
	asset, err := s.assets.Restore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("marketing asset: restore: %w", err)
	}
	return asset, nil
}
