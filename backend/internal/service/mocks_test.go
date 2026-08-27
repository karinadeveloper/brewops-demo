package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// mockUserRepository implements domain.UserRepository via injectable funcs,
// so each test wires only the behavior it needs.
type mockUserRepository struct {
	getByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
	createFunc     func(ctx context.Context, user *domain.User) error
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.getByEmailFunc(ctx, email)
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	return m.createFunc(ctx, user)
}

// mockProductRepository implements domain.ProductRepository the same way.
type mockProductRepository struct {
	createFunc         func(ctx context.Context, p *domain.Product) error
	getByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	listFunc           func(ctx context.Context, params domain.ListProductsParams) ([]domain.Product, error)
	updateFunc         func(ctx context.Context, p *domain.Product) (*domain.Product, error)
	softDeleteFunc     func(ctx context.Context, id, deletedBy uuid.UUID) error
	getTrashedByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	restoreFunc        func(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	listTrashFunc      func(ctx context.Context) ([]domain.TrashedProduct, error)
	listLowStockFunc   func(ctx context.Context) ([]domain.Product, error)
}

func (m *mockProductRepository) Create(ctx context.Context, p *domain.Product) error {
	return m.createFunc(ctx, p)
}

func (m *mockProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockProductRepository) List(ctx context.Context, params domain.ListProductsParams) ([]domain.Product, error) {
	return m.listFunc(ctx, params)
}

func (m *mockProductRepository) Update(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	return m.updateFunc(ctx, p)
}

func (m *mockProductRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return m.softDeleteFunc(ctx, id, deletedBy)
}

func (m *mockProductRepository) GetTrashedByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return m.getTrashedByIDFunc(ctx, id)
}

func (m *mockProductRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return m.restoreFunc(ctx, id)
}

func (m *mockProductRepository) ListTrash(ctx context.Context) ([]domain.TrashedProduct, error) {
	return m.listTrashFunc(ctx)
}

func (m *mockProductRepository) ListLowStock(ctx context.Context) ([]domain.Product, error) {
	return m.listLowStockFunc(ctx)
}
