package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

// SaleService implements sale creation and the soft-delete/trash/restore
// business rules on top of a domain.SaleRepository.
type SaleService struct {
	sales domain.SaleRepository
}

func NewSaleService(sales domain.SaleRepository) *SaleService {
	return &SaleService{sales: sales}
}

// computeTotalCents sums quantity*unit_price_cents across every item.
// Pure and exhaustively tested per CLAUDE.md's 100%-coverage rule — this is
// the only place total_cents is ever computed; a client-submitted total is
// never read or trusted.
func computeTotalCents(items []domain.SaleItemInput) int64 {
	var total int64
	for _, item := range items {
		total += int64(item.Quantity) * item.UnitPriceCents
	}
	return total
}

// validateSaleInput rejects structurally invalid input before any
// repository call: no items, a non-positive quantity, a negative unit
// price, or an unrecognized payment method.
func validateSaleInput(items []domain.SaleItemInput, paymentMethod string) error {
	if len(items) == 0 {
		return domain.ErrEmptySale
	}
	for _, item := range items {
		if item.Quantity <= 0 {
			return domain.ErrInvalidQuantity
		}
		if item.UnitPriceCents < 0 {
			return domain.ErrInvalidPrice
		}
	}
	if paymentMethod != "CASH" && paymentMethod != "TRANSFER" {
		return domain.ErrInvalidPaymentMethod
	}
	return nil
}

type CreateSaleInput struct {
	Items         []domain.SaleItemInput
	PaymentMethod string
	CreatedBy     uuid.UUID
	// IdempotencyKey is optional. The frontend always sends one (generated
	// when the user confirms the sale, reused across retries), but it's not
	// required at the validation level for compatibility with any other
	// caller of this endpoint. See CLAUDE.md's Business rules.
	IdempotencyKey *uuid.UUID
}

// Create validates the request, computes total_cents server-side, and
// delegates the atomic stock-check-and-decrement to the repository. Per-item
// stock availability can only be verified against live data, so that check
// — and the optimistic-concurrency guard against a concurrent sale of the
// same product — happens inside SaleRepository.Create's single transaction,
// not here.
//
// existed reports whether in.IdempotencyKey matched a sale that already
// existed (active or soft-deleted): the caller (the HTTP handler) uses this
// to respond 200 instead of 201, signaling an idempotent no-op rather than
// a freshly created sale.
func (s *SaleService) Create(ctx context.Context, in CreateSaleInput) (sale *domain.Sale, existed bool, err error) {
	if err := validateSaleInput(in.Items, in.PaymentMethod); err != nil {
		return nil, false, err
	}

	sale = &domain.Sale{
		IdempotencyKey: in.IdempotencyKey,
		TotalCents:     computeTotalCents(in.Items),
		PaymentMethod:  in.PaymentMethod,
	}
	sale.Items = make([]domain.SaleItem, len(in.Items))
	for i, item := range in.Items {
		sale.Items[i] = domain.SaleItem{
			ProductID:      item.ProductID,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	existed, err = s.sales.Create(ctx, sale, in.CreatedBy)
	if err != nil {
		return nil, false, fmt.Errorf("sale: create: %w", err)
	}
	return sale, existed, nil
}

func (s *SaleService) Get(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	return s.sales.GetByID(ctx, id)
}

type ListSalesInput struct {
	From     *time.Time
	To       *time.Time
	Page     int32
	PageSize int32
}

func (s *SaleService) List(ctx context.Context, in ListSalesInput) ([]domain.Sale, error) {
	limit := in.PageSize
	if limit <= 0 {
		limit = defaultPageSize
	}
	page := in.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	return s.sales.List(ctx, domain.ListSalesParams{
		From:   in.From,
		To:     in.To,
		Limit:  limit,
		Offset: offset,
	})
}

func (s *SaleService) SoftDelete(ctx context.Context, id, deletedBy uuid.UUID) error {
	return s.sales.SoftDelete(ctx, id, deletedBy)
}

func (s *SaleService) ListTrash(ctx context.Context) ([]domain.TrashedSale, error) {
	return s.sales.ListTrash(ctx)
}

// Restore brings a soft-deleted sale back into view without reversing its
// original stock decrement or recreating its SALE movements — see
// domain.SaleRepository.Restore. A sale, like an inventory movement, is an
// immutable historical record once created; restoring it from the trash is
// an audit-trail visibility change, not a request to re-run the sale.
func (s *SaleService) Restore(ctx context.Context, id uuid.UUID) (*domain.Sale, error) {
	sale, err := s.sales.Restore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("sale: restore: %w", err)
	}
	return sale, nil
}
