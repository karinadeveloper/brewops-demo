package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// InventoryHandler exposes /api/v1/inventory/movements/*. Every route sits
// behind middleware.Auth; as with ProductHandler, BrewOps' single-ADMIN
// model means "ADMIN only" endpoints need no extra role check.
type InventoryHandler struct {
	movements *service.InventoryService
}

func NewInventoryHandler(movements *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{movements: movements}
}

type movementResponse struct {
	ID             string  `json:"id"`
	ProductID      string  `json:"product_id"`
	Type           string  `json:"type"`
	Quantity       int32   `json:"quantity"`
	Reason         *string `json:"reason"`
	CreatedAt      string  `json:"created_at"`
	DeletedAt      *string `json:"deleted_at,omitempty"`
	DeletedByEmail *string `json:"deleted_by_email,omitempty"`
}

func toMovementResponse(m domain.InventoryMovement) movementResponse {
	resp := movementResponse{
		ID:        m.ID.String(),
		ProductID: m.ProductID.String(),
		Type:      m.Type,
		Quantity:  m.Quantity,
		Reason:    m.Reason,
		CreatedAt: m.CreatedAt.Format(timeFormat),
	}
	if m.DeletedAt != nil {
		s := m.DeletedAt.Format(timeFormat)
		resp.DeletedAt = &s
	}
	return resp
}

func toTrashedMovementResponse(m domain.TrashedInventoryMovement) movementResponse {
	resp := toMovementResponse(m.InventoryMovement)
	resp.DeletedByEmail = m.DeletedByEmail
	return resp
}

type createMovementRequest struct {
	ProductID string  `json:"product_id"`
	Type      string  `json:"type"`
	Quantity  int32   `json:"quantity"`
	Reason    *string `json:"reason"`
}

func (h *InventoryHandler) Create(c *fiber.Ctx) error {
	var req createMovementRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	movement, err := h.movements.CreateManual(c.Context(), service.CreateMovementInput{
		ProductID: productID,
		Type:      req.Type,
		Quantity:  req.Quantity,
		Reason:    req.Reason,
		CreatedBy: userID,
	})
	if err != nil {
		return mapMovementError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(toMovementResponse(*movement))
}

func (h *InventoryHandler) List(c *fiber.Ctx) error {
	var productID *uuid.UUID
	if v := c.Query("product_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
		}
		productID = &id
	}

	from, err := parseQueryTime(c, "from")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid from date")
	}
	to, err := parseQueryTime(c, "to")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid to date")
	}

	movements, err := h.movements.List(c.Context(), service.ListMovementsInput{
		ProductID: productID,
		From:      from,
		To:        to,
		Page:      int32(c.QueryInt("page", 1)),
		PageSize:  int32(c.QueryInt("page_size", 20)),
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list inventory movements")
	}

	resp := make([]movementResponse, len(movements))
	for i, m := range movements {
		resp[i] = toMovementResponse(m)
	}
	return c.JSON(resp)
}

func (h *InventoryHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid movement id")
	}

	userID, err := currentUserID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.movements.SoftDelete(c.Context(), id, userID); err != nil {
		return mapMovementError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *InventoryHandler) Trash(c *fiber.Ctx) error {
	trashed, err := h.movements.ListTrash(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list trash")
	}

	resp := make([]movementResponse, len(trashed))
	for i, m := range trashed {
		resp[i] = toTrashedMovementResponse(m)
	}
	return c.JSON(resp)
}

func (h *InventoryHandler) Restore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid movement id")
	}

	movement, err := h.movements.Restore(c.Context(), id)
	if err != nil {
		return mapMovementError(err)
	}
	return c.JSON(toMovementResponse(*movement))
}

// parseQueryTime parses an RFC3339 query param, returning nil (not an
// error) when the param is absent.
func parseQueryTime(c *fiber.Ctx, name string) (*time.Time, error) {
	v := c.Query(name)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func mapMovementError(err error) error {
	switch {
	case errors.Is(err, domain.ErrMovementNotFound), errors.Is(err, domain.ErrProductNotFound):
		return fiber.NewError(fiber.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrManualSaleMovement), errors.Is(err, domain.ErrInvalidMovementType), errors.Is(err, domain.ErrInvalidQuantity):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInsufficientStock):
		return fiber.NewError(fiber.StatusConflict, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "unexpected error")
	}
}
