package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// InventorySuggestionHandler exposes POST /api/v1/inventory/suggest. It
// never persists anything — the response is a list of suggestions the
// frontend must let the user review and confirm through the existing
// POST /inventory/movements before anything is written.
type InventorySuggestionHandler struct {
	suggestions *service.InventorySuggestionService
}

func NewInventorySuggestionHandler(suggestions *service.InventorySuggestionService) *InventorySuggestionHandler {
	return &InventorySuggestionHandler{suggestions: suggestions}
}

type suggestRequest struct {
	Text string `json:"text"`
}

type suggestedMovementResponse struct {
	ProductMentioned string  `json:"product_mentioned"`
	ProductMatch     *string `json:"product_match"`
	Type             string  `json:"type"`
	Quantity         int32   `json:"quantity"`
	UnitCostCents    int64   `json:"unit_cost_cents"`
}

func (h *InventorySuggestionHandler) Suggest(c *fiber.Ctx) error {
	var req suggestRequest
	if err := c.BodyParser(&req); err != nil || req.Text == "" {
		return fiber.NewError(fiber.StatusBadRequest, "text is required")
	}

	suggestions, err := h.suggestions.Suggest(c.Context(), req.Text)
	if err != nil {
		if errors.Is(err, domain.ErrAISuggestionFailed) {
			return fiber.NewError(fiber.StatusBadGateway, "AI suggestion service is unavailable, try again shortly")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "unexpected error")
	}

	resp := make([]suggestedMovementResponse, len(suggestions))
	for i, s := range suggestions {
		var match *string
		if s.ProductMatch != nil {
			id := s.ProductMatch.String()
			match = &id
		}
		resp[i] = suggestedMovementResponse{
			ProductMentioned: s.ProductMentioned,
			ProductMatch:     match,
			Type:             s.Type,
			Quantity:         s.Quantity,
			UnitCostCents:    s.UnitCostCents,
		}
	}
	return c.JSON(resp)
}
