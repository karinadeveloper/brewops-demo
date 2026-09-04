package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

const defaultPageSize int32 = 20

// ProductService implements the product CRUD and soft-delete/trash/restore
// business rules on top of a domain.ProductRepository.
type ProductService struct {
	products domain.ProductRepository
}

func NewProductService(products domain.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

// validatePricing is the single source of truth for the "no negative
// prices/stock" rule — pure, deterministic, and exhaustively tested, as
// required for any price/inventory calculation function in this project.
func validatePricing(salePriceCents, costCents int64, currentStock, minStock int32) error {
	if salePriceCents < 0 || costCents < 0 {
		return domain.ErrInvalidPrice
	}
	if currentStock < 0 || minStock < 0 {
		return domain.ErrInvalidStock
	}
	return nil
}

type CreateProductInput struct {
	Name           string
	Category       string
	SalePriceCents int64
	CostCents      int64
	CurrentStock   int32
	MinStock       int32
	ImageURL       *string
	CreatedBy      uuid.UUID
}

func (s *ProductService) Create(ctx context.Context, in CreateProductInput) (*domain.Product, error) {
	if err := validatePricing(in.SalePriceCents, in.CostCents, in.CurrentStock, in.MinStock); err != nil {
		return nil, err
	}

	p := &domain.Product{
		Name:           in.Name,
		Category:       in.Category,
		SalePriceCents: in.SalePriceCents,
		CostCents:      in.CostCents,
		CurrentStock:   in.CurrentStock,
		MinStock:       in.MinStock,
		ImageURL:       in.ImageURL,
		UpdatedBy:      &in.CreatedBy,
	}
	if err := s.products.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("product: create: %w", err)
	}
	return p, nil
}

func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return s.products.GetByID(ctx, id)
}

type ListProductsInput struct {
	Category *string
	Page     int32
	PageSize int32
}

func (s *ProductService) List(ctx context.Context, in ListProductsInput) ([]domain.Product, error) {
	limit := in.PageSize
	if limit <= 0 {
		limit = defaultPageSize
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	return s.products.List(ctx, domain.ListProductsParams{
		Category: in.Category,
		Limit:    limit,
		Offset:   offset,
	})
}

type UpdateProductInput struct {
	ID             uuid.UUID
	Version        int32
	Name           string
	Category       string
	SalePriceCents int64
	CostCents      int64
	CurrentStock   int32
	MinStock       int32
	ImageURL       *string
	UpdatedBy      uuid.UUID
}

// Update applies a PATCH under optimistic concurrency: the caller's Version
// must still match the row's current version, or domain.ErrOptimisticLockConflict
// is returned. The product is fetched first so a genuinely missing product
// reports domain.ErrProductNotFound instead of being conflated with a
// version race.
func (s *ProductService) Update(ctx context.Context, in UpdateProductInput) (*domain.Product, error) {
	if err := validatePricing(in.SalePriceCents, in.CostCents, in.CurrentStock, in.MinStock); err != nil {
		return nil, err
	}

	if _, err := s.products.GetByID(ctx, in.ID); err != nil {
		return nil, err
	}

	p := &domain.Product{
		ID:             in.ID,
		Version:        in.Version,
		Name:           in.Name,
		Category:       in.Category,
		SalePriceCents: in.SalePriceCents,
		CostCents:      in.CostCents,
		CurrentStock:   in.CurrentStock,
		MinStock:       in.MinStock,
		ImageURL:       in.ImageURL,
		UpdatedBy:      &in.UpdatedBy,
	}
	return s.products.Update(ctx, p)
}

func (s *ProductService) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return s.products.SoftDelete(ctx, id, deletedBy)
}

func (s *ProductService) ListTrash(ctx context.Context) ([]domain.TrashedProduct, error) {
	return s.products.ListTrash(ctx)
}

// Restore re-checks current_stock >= 0 before clearing deleted_at/deleted_by.
// This guards against the case (which shouldn't be able to happen, but is
// defended against anyway) where an inventory movement altered stock while
// the product was soft-deleted, leaving it in an invalid state that would
// otherwise silently reappear in the active catalog. If the check fails,
// domain.ErrInvalidRestoreState is returned wrapping the current stock value
// so the caller can surface a clear, actionable message ("adjust inventory
// first") instead of a generic failure.
func (s *ProductService) Restore(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, err := s.products.GetTrashedByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if p.CurrentStock < 0 {
		return nil, fmt.Errorf("%w: current stock is %d, adjust inventory first", domain.ErrInvalidRestoreState, p.CurrentStock)
	}

	return s.products.Restore(ctx, id)
}

func (s *ProductService) ListLowStock(ctx context.Context) ([]domain.Product, error) {
	return s.products.ListLowStock(ctx)
}
