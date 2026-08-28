package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// MarketingAssetHandler exposes /api/v1/marketing/assets/*. Every route
// sits behind middleware.Auth; as with ProductHandler, BrewOps'
// single-ADMIN model means "ADMIN only" endpoints need no extra role check.
type MarketingAssetHandler struct {
	assets *service.MarketingAssetService
}

func NewMarketingAssetHandler(assets *service.MarketingAssetService) *MarketingAssetHandler {
	return &MarketingAssetHandler{assets: assets}
}

type marketingAssetResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ImageURL       string  `json:"image_url"`
	Type           string  `json:"type"`
	CreatedAt      string  `json:"created_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	DeletedByEmail *string `json:"deleted_by_email,omitempty"`
}

func toMarketingAssetResponse(a domain.MarketingAsset) marketingAssetResponse {
	resp := marketingAssetResponse{
		ID:        a.ID.String(),
		Name:      a.Name,
		ImageURL:  a.ImageURL,
		Type:      a.Type,
		CreatedAt: a.CreatedAt.Format(timeFormat),
	}
	if a.DeletedAt != nil {
		d := a.DeletedAt.Format(timeFormat)
		resp.DeletedAt = &d
	}
	return resp
}

func toTrashedMarketingAssetResponse(a domain.TrashedMarketingAsset) marketingAssetResponse {
	resp := toMarketingAssetResponse(a.MarketingAsset)
	resp.DeletedByEmail = a.DeletedByEmail
	return resp
}

func (h *MarketingAssetHandler) Create(c *fiber.Ctx) error {
	data, contentType, err := readImageUpload(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	asset, err := h.assets.Create(c.Context(), service.CreateMarketingAssetInput{
		Name:        c.FormValue("name"),
		Type:        c.FormValue("type"),
		ImageData:   data,
		ContentType: contentType,
		CreatedBy:   userID,
	})
	if err != nil {
		return mapMarketingAssetError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(toMarketingAssetResponse(*asset))
}

func (h *MarketingAssetHandler) List(c *fiber.Ctx) error {
	assets, err := h.assets.List(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list marketing assets")
	}

	resp := make([]marketingAssetResponse, len(assets))
	for i, a := range assets {
		resp[i] = toMarketingAssetResponse(a)
	}
	return c.JSON(resp)
}

func (h *MarketingAssetHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid marketing asset id")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.assets.SoftDelete(c.Context(), id, userID); err != nil {
		return mapMarketingAssetError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *MarketingAssetHandler) Trash(c *fiber.Ctx) error {
	trashed, err := h.assets.ListTrash(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list trash")
	}

	resp := make([]marketingAssetResponse, len(trashed))
	for i, a := range trashed {
		resp[i] = toTrashedMarketingAssetResponse(a)
	}
	return c.JSON(resp)
}

func (h *MarketingAssetHandler) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid marketing asset id")
	}

	asset, err := h.assets.Restore(c.Context(), id)
	if err != nil {
		return mapMarketingAssetError(err)
	}
	return c.JSON(toMarketingAssetResponse(*asset))
}

func mapMarketingAssetError(err error) error {
	switch {
	case errors.Is(err, domain.ErrMarketingAssetNotFound):
		return fiber.NewError(fiber.StatusNotFound, "marketing asset not found")
	case errors.Is(err, domain.ErrInvalidMarketingAssetType), errors.Is(err, domain.ErrInvalidImageContentType):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "unexpected error")
	}
}
