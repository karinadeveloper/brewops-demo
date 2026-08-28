package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Marketing asset types — the only valid values for MarketingAsset.Type.
const (
	MarketingAssetTypeProduct   = "PRODUCT"
	MarketingAssetTypePromotion = "PROMOTION"
)

var (
	// ErrMarketingAssetNotFound is returned when no marketing asset matches
	// the lookup in the scope being queried (active or trashed).
	ErrMarketingAssetNotFound = errors.New("marketing asset not found")
	// ErrInvalidMarketingAssetType is returned for any type outside
	// PRODUCT/PROMOTION.
	ErrInvalidMarketingAssetType = errors.New("marketing asset type must be PRODUCT or PROMOTION")
)

// MarketingAsset mirrors the marketing_assets table — an image the owners
// post on WhatsApp, either a product photo or a promotional graphic.
type MarketingAsset struct {
	ID        uuid.UUID
	Name      string
	ImageURL  string
	Type      string
	CreatedAt time.Time
	CreatedBy *uuid.UUID
	DeletedAt *time.Time
	DeletedBy *uuid.UUID
}

// TrashedMarketingAsset is a soft-deleted MarketingAsset enriched with the
// email of the user who deleted it, for the trash view.
type TrashedMarketingAsset struct {
	MarketingAsset
	DeletedByEmail *string
}

// MarketingAssetRepository is the persistence boundary the marketing asset
// service depends on. Soft-deleting or restoring a row here never touches
// the underlying file in StorageClient — see
// service.MarketingAssetService for why.
type MarketingAssetRepository interface {
	Create(ctx context.Context, a *MarketingAsset) error
	List(ctx context.Context) ([]MarketingAsset, error)
	SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) (*MarketingAsset, error)
	ListTrash(ctx context.Context) ([]TrashedMarketingAsset, error)
}
