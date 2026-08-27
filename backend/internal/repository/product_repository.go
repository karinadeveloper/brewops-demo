package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// ProductRepository adapts the sqlc-generated Queries to domain.ProductRepository.
type ProductRepository struct {
	q *Queries
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{q: New(pool)}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	row, err := r.q.CreateProduct(ctx, CreateProductParams{
		Name:           p.Name,
		Category:       p.Category,
		SalePriceCents: p.SalePriceCents,
		CostCents:      p.CostCents,
		CurrentStock:   p.CurrentStock,
		MinStock:       p.MinStock,
		ImageUrl:       toText(p.ImageURL),
		UpdatedBy:      toNullUUID(p.UpdatedBy),
	})
	if err != nil {
		return err
	}
	*p = *toDomainProduct(row)
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	row, err := r.q.GetProductByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return toDomainProduct(row), nil
}

func (r *ProductRepository) GetTrashedByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	row, err := r.q.GetTrashedProductByID(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return toDomainProduct(row), nil
}

func (r *ProductRepository) List(ctx context.Context, params domain.ListProductsParams) ([]domain.Product, error) {
	rows, err := r.q.ListProducts(ctx, ListProductsParams{
		Category: toText(params.Category),
		Limit:    params.Limit,
		Offset:   params.Offset,
	})
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, len(rows))
	for i, row := range rows {
		products[i] = *toDomainProduct(row)
	}
	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	row, err := r.q.UpdateProduct(ctx, UpdateProductParams{
		ID:             toUUID(p.ID),
		Name:           p.Name,
		Category:       p.Category,
		SalePriceCents: p.SalePriceCents,
		CostCents:      p.CostCents,
		CurrentStock:   p.CurrentStock,
		MinStock:       p.MinStock,
		ImageUrl:       toText(p.ImageURL),
		UpdatedBy:      toNullUUID(p.UpdatedBy),
		Version:        p.Version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOptimisticLockConflict
		}
		return nil, err
	}
	return toDomainProduct(row), nil
}

func (r *ProductRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	rowsAffected, err := r.q.SoftDeleteProduct(ctx, SoftDeleteProductParams{
		ID:        toUUID(id),
		DeletedBy: toNullUUID(&deletedBy),
	})
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrProductNotFound
	}
	return nil
}

func (r *ProductRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	row, err := r.q.RestoreProduct(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, err
	}
	return toDomainProduct(row), nil
}

func (r *ProductRepository) ListTrash(ctx context.Context) ([]domain.TrashedProduct, error) {
	rows, err := r.q.ListTrashedProducts(ctx)
	if err != nil {
		return nil, err
	}
	trashed := make([]domain.TrashedProduct, len(rows))
	for i, row := range rows {
		trashed[i] = domain.TrashedProduct{
			Product: domain.Product{
				ID:             fromUUID(row.ID),
				Name:           row.Name,
				Category:       row.Category,
				SalePriceCents: row.SalePriceCents,
				CostCents:      row.CostCents,
				CurrentStock:   row.CurrentStock,
				MinStock:       row.MinStock,
				ImageURL:       fromText(row.ImageUrl),
				Version:        row.Version,
				CreatedAt:      fromTimestamptz(row.CreatedAt),
				UpdatedAt:      fromTimestamptz(row.UpdatedAt),
				UpdatedBy:      fromNullUUID(row.UpdatedBy),
				DeletedAt:      fromNullTimestamptz(row.DeletedAt),
				DeletedBy:      fromNullUUID(row.DeletedBy),
			},
			DeletedByEmail: fromText(row.DeletedByEmail),
		}
	}
	return trashed, nil
}

func (r *ProductRepository) ListLowStock(ctx context.Context) ([]domain.Product, error) {
	rows, err := r.q.ListLowStockProducts(ctx)
	if err != nil {
		return nil, err
	}
	products := make([]domain.Product, len(rows))
	for i, row := range rows {
		products[i] = *toDomainProduct(row)
	}
	return products, nil
}

func toDomainProduct(row Product) *domain.Product {
	return &domain.Product{
		ID:             fromUUID(row.ID),
		Name:           row.Name,
		Category:       row.Category,
		SalePriceCents: row.SalePriceCents,
		CostCents:      row.CostCents,
		CurrentStock:   row.CurrentStock,
		MinStock:       row.MinStock,
		ImageURL:       fromText(row.ImageUrl),
		Version:        row.Version,
		CreatedAt:      fromTimestamptz(row.CreatedAt),
		UpdatedAt:      fromTimestamptz(row.UpdatedAt),
		UpdatedBy:      fromNullUUID(row.UpdatedBy),
		DeletedAt:      fromNullTimestamptz(row.DeletedAt),
		DeletedBy:      fromNullUUID(row.DeletedBy),
	}
}
