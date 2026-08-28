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

// mockInventoryMovementRepository implements domain.InventoryMovementRepository
// the same way.
type mockInventoryMovementRepository struct {
	createManualFunc   func(ctx context.Context, m *domain.InventoryMovement, stockDelta int32) error
	getByIDFunc        func(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error)
	listFunc           func(ctx context.Context, params domain.ListMovementsParams) ([]domain.InventoryMovement, error)
	softDeleteFunc     func(ctx context.Context, id, deletedBy uuid.UUID) error
	getTrashedByIDFunc func(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error)
	restoreFunc        func(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error)
	listTrashFunc      func(ctx context.Context) ([]domain.TrashedInventoryMovement, error)
}

func (m *mockInventoryMovementRepository) CreateManual(ctx context.Context, movement *domain.InventoryMovement, stockDelta int32) error {
	return m.createManualFunc(ctx, movement, stockDelta)
}

func (m *mockInventoryMovementRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockInventoryMovementRepository) List(ctx context.Context, params domain.ListMovementsParams) ([]domain.InventoryMovement, error) {
	return m.listFunc(ctx, params)
}

func (m *mockInventoryMovementRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return m.softDeleteFunc(ctx, id, deletedBy)
}

func (m *mockInventoryMovementRepository) GetTrashedByID(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	return m.getTrashedByIDFunc(ctx, id)
}

func (m *mockInventoryMovementRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.InventoryMovement, error) {
	return m.restoreFunc(ctx, id)
}

func (m *mockInventoryMovementRepository) ListTrash(ctx context.Context) ([]domain.TrashedInventoryMovement, error) {
	return m.listTrashFunc(ctx)
}

// mockSaleRepository implements domain.SaleRepository the same way.
type mockSaleRepository struct {
	createFunc     func(ctx context.Context, sale *domain.Sale, createdBy uuid.UUID) error
	getByIDFunc    func(ctx context.Context, id uuid.UUID) (*domain.Sale, error)
	listFunc       func(ctx context.Context, params domain.ListSalesParams) ([]domain.Sale, error)
	softDeleteFunc func(ctx context.Context, id, deletedBy uuid.UUID) error
	restoreFunc    func(ctx context.Context, id uuid.UUID) (*domain.Sale, error)
	listTrashFunc  func(ctx context.Context) ([]domain.TrashedSale, error)
}

func (m *mockSaleRepository) Create(ctx context.Context, sale *domain.Sale, createdBy uuid.UUID) error {
	return m.createFunc(ctx, sale, createdBy)
}

func (m *mockSaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	return m.getByIDFunc(ctx, id)
}

func (m *mockSaleRepository) List(ctx context.Context, params domain.ListSalesParams) ([]domain.Sale, error) {
	return m.listFunc(ctx, params)
}

func (m *mockSaleRepository) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return m.softDeleteFunc(ctx, id, deletedBy)
}

func (m *mockSaleRepository) Restore(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	return m.restoreFunc(ctx, id)
}

func (m *mockSaleRepository) ListTrash(ctx context.Context) ([]domain.TrashedSale, error) {
	return m.listTrashFunc(ctx)
}
