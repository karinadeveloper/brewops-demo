package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// MarketingAssetRepository adapts the sqlc-generated Queries to
// domain.MarketingAssetRepository.
type MarketingAssetRepository struct {
	q *Queries
}

func NewMarketingAssetRepository(pool *pgxpool.Pool) *MarketingAssetRepository {
	return &MarketingAssetRepository{q: New(pool)}
}

func (r *MarketingAssetRepository) Create(ctx context.Context, a *domain.MarketingAsset) error {
	row, err := r.q.CreateMarketingAsset(ctx, CreateMarketingAssetParams{
		Name:      a.Name,
		ImageUrl:  a.ImageURL,
		Type:      a.Type,
		CreatedBy: toNullUUID(a.CreatedBy),
	})
	if err != nil {
		return err
	}
	*a = *toDomainMarketingAsset(row)
	return nil
}

func (r *MarketingAssetRepository) List(ctx context.Context) ([]domain.MarketingAsset, error) {
	rows, err := r.q.ListMarketingAssets(ctx)
	if err != nil {
		return nil, err
	}
	assets := make([]domain.MarketingAsset, len(rows))
	for i, row := range rows {
		assets[i] = *toDomainMarketingAsset(row)
	}
	return assets, nil
}

func (r *MarketingAssetRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	rowsAffected, err := r.q.SoftDeleteMarketingAsset(ctx, SoftDeleteMarketingAssetParams{
		ID:        toUUID(id),
		DeletedBy: toNullUUID(&deletedBy),
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrMarketingAssetNotFound
	}
	return nil
}

// Restore clears deleted_at/deleted_by only. It never re-uploads or
// otherwise touches the underlying file — the file was never deleted by
// SoftDelete in the first place, so ImageURL already points at a live
// object.
func (r *MarketingAssetRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.MarketingAsset, error) {
	row, err := r.q.RestoreMarketingAsset(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMarketingAssetNotFound
		}
		return nil, err
	}
	return toDomainMarketingAsset(row), nil
}

func (r *MarketingAssetRepository) ListTrash(ctx context.Context) ([]domain.TrashedMarketingAsset, error) {
	rows, err := r.q.ListTrashedMarketingAssets(ctx)
	if err != nil {
		return nil, err
	}
	trashed := make([]domain.TrashedMarketingAsset, len(rows))
	for i, row := range rows {
		trashed[i] = domain.TrashedMarketingAsset{
			MarketingAsset: domain.MarketingAsset{
				ID:        fromUUID(row.ID),
				Name:      row.Name,
				ImageURL:  row.ImageUrl,
				Type:      row.Type,
				CreatedAt: fromTimestamptz(row.CreatedAt),
				CreatedBy: fromNullUUID(row.CreatedBy),
				DeletedAt: fromNullTimestamptz(row.DeletedAt),
				DeletedBy: fromNullUUID(row.DeletedBy),
			},
			DeletedByEmail: fromText(row.DeletedByEmail),
		}
	}
	return trashed, nil
}

func toDomainMarketingAsset(row MarketingAsset) *domain.MarketingAsset {
	return &domain.MarketingAsset{
		ID:        fromUUID(row.ID),
		Name:      row.Name,
		ImageURL:  row.ImageUrl,
		Type:      row.Type,
		CreatedAt: fromTimestamptz(row.CreatedAt),
		CreatedBy: fromNullUUID(row.CreatedBy),
		DeletedAt: fromNullTimestamptz(row.DeletedAt),
		DeletedBy: fromNullUUID(row.DeletedBy),
	}
}
