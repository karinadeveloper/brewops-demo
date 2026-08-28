package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// ReportHandler exposes /api/v1/reports/*, every route sits behind
// middleware.Auth like the rest of the API.
type ReportHandler struct {
	reports *service.ReportService
}

func NewReportHandler(reports *service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

type revenuePointResponse struct {
	Day        *string `json:"day,omitempty"`
	TotalCents int64   `json:"total_cents"`
}

func (h *ReportHandler) Revenue(c *fiber.Ctx) error {
	from, err := parseQueryTime(c, "from")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid from date")
	}
	to, err := parseQueryTime(c, "to")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid to date")
	}

	points, err := h.reports.Revenue(c.Context(), service.RevenueInput{
		From:       from,
		To:         to,
		GroupByDay: c.Query("group_by") == "day",
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to compute revenue")
	}

	resp := make([]revenuePointResponse, len(points))
	for i, p := range points {
		r := revenuePointResponse{TotalCents: p.TotalCents}
		if p.Day != nil {
			d := p.Day.Format("2006-01-02")
			r.Day = &d
		}
		resp[i] = r
	}
	return c.JSON(resp)
}

type topProductResponse struct {
	ProductID    string `json:"product_id"`
	Name         string `json:"name"`
	QuantitySold int32  `json:"quantity_sold"`
	RevenueCents int64  `json:"revenue_cents"`
}

func (h *ReportHandler) TopProducts(c *fiber.Ctx) error {
	from, err := parseQueryTime(c, "from")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid from date")
	}
	to, err := parseQueryTime(c, "to")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid to date")
	}

	products, err := h.reports.TopProducts(c.Context(), service.TopProductsInput{
		From:   from,
		To:     to,
		SortBy: c.Query("sort"),
		Limit:  int32(c.QueryInt("limit", 0)),
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidSortBy) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to compute top products")
	}

	resp := make([]topProductResponse, len(products))
	for i, p := range products {
		resp[i] = topProductResponse{
			ProductID:    p.ProductID.String(),
			Name:         p.Name,
			QuantitySold: p.QuantitySold,
			RevenueCents: p.RevenueCents,
		}
	}
	return c.JSON(resp)
}

type inventoryValueResponse struct {
	TotalValueCents int64 `json:"total_value_cents"`
}

func (h *ReportHandler) InventoryValue(c *fiber.Ctx) error {
	total, err := h.reports.InventoryValue(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to compute inventory value")
	}
	return c.JSON(inventoryValueResponse{TotalValueCents: total})
}
