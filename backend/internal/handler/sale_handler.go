package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// SaleHandler exposes /api/v1/sales/*. Every route sits behind
// middleware.Auth; as with ProductHandler, BrewOps' single-ADMIN model
// means "ADMIN only" endpoints need no extra role check.
type SaleHandler struct {
	sales *service.SaleService
}

func NewSaleHandler(sales *service.SaleService) *SaleHandler {
	return &SaleHandler{sales: sales}
}

type saleItemResponse struct {
	ProductID      string `json:"product_id"`
	Quantity       int32  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

type saleResponse struct {
	ID             string             `json:"id"`
	Items          []saleItemResponse `json:"items"`
	TotalCents     int64              `json:"total_cents"`
	PaymentMethod  string             `json:"payment_method"`
	CreatedAt      string             `json:"created_at"`
	DeletedAt      *string            `json:"deleted_at,omitempty"`
	DeletedByEmail *string            `json:"deleted_by_email,omitempty"`
}

func toSaleResponse(s domain.Sale) saleResponse {
	items := make([]saleItemResponse, len(s.Items))
	for i, item := range s.Items {
		items[i] = saleItemResponse{
			ProductID:      item.ProductID.String(),
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}
	resp := saleResponse{
		ID:            s.ID.String(),
		Items:         items,
		TotalCents:    s.TotalCents,
		PaymentMethod: s.PaymentMethod,
		CreatedAt:     s.CreatedAt.Format(timeFormat),
	}
	if s.DeletedAt != nil {
		d := s.DeletedAt.Format(timeFormat)
		resp.DeletedAt = &d
	}
	return resp
}

func toTrashedSaleResponse(s domain.TrashedSale) saleResponse {
	resp := toSaleResponse(s.Sale)
	resp.DeletedByEmail = s.DeletedByEmail
	return resp
}

type createSaleItemRequest struct {
	ProductID      string `json:"product_id"`
	Quantity       int32  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

type createSaleRequest struct {
	Items          []createSaleItemRequest `json:"items"`
	PaymentMethod  string                  `json:"payment_method"`
	IdempotencyKey *string                 `json:"idempotency_key,omitempty"`
}

func (h *SaleHandler) Create(c *fiber.Ctx) error {
	var req createSaleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	items := make([]domain.SaleItemInput, len(req.Items))
	for i, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid product_id in items")
		}
		items[i] = domain.SaleItemInput{
			ProductID:      productID,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
		}
	}

	var idempotencyKey *uuid.UUID
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		key, err := uuid.Parse(*req.IdempotencyKey)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid idempotency_key")
		}
		idempotencyKey = &key
	}

	sale, existed, err := h.sales.Create(c.Context(), service.CreateSaleInput{
		Items:          items,
		PaymentMethod:  req.PaymentMethod,
		CreatedBy:      userID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return mapSaleError(err)
	}

	// existed=true means idempotencyKey matched a sale already created by a
	// prior attempt — nothing new was persisted, so this is a 200 (an
	// idempotent no-op), not a 201 (a fresh creation).
	status := fiber.StatusCreated
	if existed {
		status = fiber.StatusOK
	}
	return c.Status(status).JSON(toSaleResponse(*sale))
}

func (h *SaleHandler) List(c *fiber.Ctx) error {
	from, err := parseQueryTime(c, "from")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid from date")
	}
	to, err := parseQueryTime(c, "to")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid to date")
	}

	sales, err := h.sales.List(c.Context(), service.ListSalesInput{
		From:     from,
		To:       to,
		Page:     int32(c.QueryInt("page", 1)),
		PageSize: int32(c.QueryInt("page_size", 20)),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list sales")
	}

	resp := make([]saleResponse, len(sales))
	for i, s := range sales {
		resp[i] = toSaleResponse(s)
	}
	return c.JSON(resp)
}

func (h *SaleHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sale id")
	}

	sale, err := h.sales.Get(c.Context(), id)
	if err != nil {
		return mapSaleError(err)
	}
	return c.JSON(toSaleResponse(*sale))
}

func (h *SaleHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sale id")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.sales.SoftDelete(c.Context(), id, userID); err != nil {
		return mapSaleError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *SaleHandler) Trash(c *fiber.Ctx) error {
	trashed, err := h.sales.ListTrash(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list trash")
	}

	resp := make([]saleResponse, len(trashed))
	for i, s := range trashed {
		resp[i] = toTrashedSaleResponse(s)
	}
	return c.JSON(resp)
}

func (h *SaleHandler) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid sale id")
	}

	sale, err := h.sales.Restore(c.Context(), id)
	if err != nil {
		return mapSaleError(err)
	}
	return c.JSON(toSaleResponse(*sale))
}

func mapSaleError(err error) error {
	switch {
	case errors.Is(err, domain.ErrSaleNotFound), errors.Is(err, domain.ErrProductNotFound):
		return fiber.NewError(fiber.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrEmptySale), errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidPrice), errors.Is(err, domain.ErrInvalidPaymentMethod):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrOptimisticLockConflict), errors.Is(err, domain.ErrInsufficientStock):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "unexpected error")
	}
}
