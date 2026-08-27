package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// ProductHandler exposes /api/v1/products/*. Every route here sits behind
// middleware.Auth, and since BrewOps has no multi-tenant roles beyond the
// single ADMIN, "ADMIN only" endpoints (trash, restore) need no extra
// role check — any authenticated request already is the admin.
type ProductHandler struct {
	products *service.ProductService
}

func NewProductHandler(products *service.ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

type productResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	SalePriceCents int64   `json:"sale_price_cents"`
	CostCents      int64   `json:"cost_cents"`
	CurrentStock   int32   `json:"current_stock"`
	MinStock       int32   `json:"min_stock"`
	ImageURL       *string `json:"image_url"`
	Version        int32   `json:"version"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	DeletedByEmail *string `json:"deleted_by_email,omitempty"`
}

func toProductResponse(p domain.Product) productResponse {
	resp := productResponse{
		ID:             p.ID.String(),
		Name:           p.Name,
		Category:       p.Category,
		SalePriceCents: p.SalePriceCents,
		CostCents:      p.CostCents,
		CurrentStock:   p.CurrentStock,
		MinStock:       p.MinStock,
		ImageURL:       p.ImageURL,
		Version:        p.Version,
		CreatedAt:      p.CreatedAt.Format(timeFormat),
		UpdatedAt:      p.UpdatedAt.Format(timeFormat),
	}
	if p.DeletedAt != nil {
		s := p.DeletedAt.Format(timeFormat)
		resp.DeletedAt = &s
	}
	return resp
}

func toTrashedProductResponse(p domain.TrashedProduct) productResponse {
	resp := toProductResponse(p.Product)
	resp.DeletedByEmail = p.DeletedByEmail
	return resp
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

type createProductRequest struct {
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	SalePriceCents int64   `json:"sale_price_cents"`
	CostCents      int64   `json:"cost_cents"`
	CurrentStock   int32   `json:"current_stock"`
	MinStock       int32   `json:"min_stock"`
	ImageURL       *string `json:"image_url"`
}

func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var req createProductRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	product, err := h.products.Create(c.Context(), service.CreateProductInput{
		Name:           req.Name,
		Category:       req.Category,
		SalePriceCents: req.SalePriceCents,
		CostCents:      req.CostCents,
		CurrentStock:   req.CurrentStock,
		MinStock:       req.MinStock,
		ImageURL:       req.ImageURL,
		CreatedBy:      userID,
	})
	if err != nil {
		return mapProductError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(toProductResponse(*product))
}

func (h *ProductHandler) List(c *fiber.Ctx) error {
	var category *string
	if v := c.Query("category"); v != "" {
		category = &v
	}

	products, err := h.products.List(c.Context(), service.ListProductsInput{
		Category: category,
		Page:     int32(c.QueryInt("page", 1)),
		PageSize: int32(c.QueryInt("page_size", 20)),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list products")
	}

	resp := make([]productResponse, len(products))
	for i, p := range products {
		resp[i] = toProductResponse(p)
	}
	return c.JSON(resp)
}

func (h *ProductHandler) Get(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product id")
	}

	product, err := h.products.Get(c.Context(), id)
	if err != nil {
		return mapProductError(err)
	}
	return c.JSON(toProductResponse(*product))
}

type updateProductRequest struct {
	Version        int32   `json:"version"`
	Name           string  `json:"name"`
	Category       string  `json:"category"`
	SalePriceCents int64   `json:"sale_price_cents"`
	CostCents      int64   `json:"cost_cents"`
	CurrentStock   int32   `json:"current_stock"`
	MinStock       int32   `json:"min_stock"`
	ImageURL       *string `json:"image_url"`
}

func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product id")
	}

	var req updateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	product, err := h.products.Update(c.Context(), service.UpdateProductInput{
		ID:             id,
		Version:        req.Version,
		Name:           req.Name,
		Category:       req.Category,
		SalePriceCents: req.SalePriceCents,
		CostCents:      req.CostCents,
		CurrentStock:   req.CurrentStock,
		MinStock:       req.MinStock,
		ImageURL:       req.ImageURL,
		UpdatedBy:      userID,
	})
	if err != nil {
		return mapProductError(err)
	}

	return c.JSON(toProductResponse(*product))
}

func (h *ProductHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product id")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.products.SoftDelete(c.Context(), id, userID); err != nil {
		return mapProductError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ProductHandler) Trash(c *fiber.Ctx) error {
	trashed, err := h.products.ListTrash(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list trash")
	}

	resp := make([]productResponse, len(trashed))
	for i, p := range trashed {
		resp[i] = toTrashedProductResponse(p)
	}
	return c.JSON(resp)
}

func (h *ProductHandler) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product id")
	}

	product, err := h.products.Restore(c.Context(), id)
	if err != nil {
		return mapProductError(err)
	}
	return c.JSON(toProductResponse(*product))
}

func (h *ProductHandler) LowStock(c *fiber.Ctx) error {
	products, err := h.products.ListLowStock(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list low-stock products")
	}

	resp := make([]productResponse, len(products))
	for i, p := range products {
		resp[i] = toProductResponse(p)
	}
	return c.JSON(resp)
}

func mapProductError(err error) error {
	switch {
	case errors.Is(err, domain.ErrProductNotFound):
		return fiber.NewError(fiber.StatusNotFound, "product not found")
	case errors.Is(err, domain.ErrInvalidPrice), errors.Is(err, domain.ErrInvalidStock):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrOptimisticLockConflict):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidRestoreState):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "unexpected error")
	}
}
