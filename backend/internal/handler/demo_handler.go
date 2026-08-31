package handler

import (
	"crypto/subtle"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// DemoResetTokenHeader carries the shared secret POST
// /api/v1/admin/demo-reset requires instead of a normal JWT — the caller is
// an external trigger (Cloud Scheduler) with no user session to present a
// bearer token for. Demo-only — see CLAUDE.md's "DEMO MODE" section; this
// route is only ever registered when config.Config.DemoMode is true (see
// cmd/server/main.go), so a request against a non-demo deployment gets a
// plain 404 rather than reaching this handler at all.
const DemoResetTokenHeader = "X-Demo-Reset-Token"

// DemoHandler exposes POST /api/v1/admin/demo-reset.
type DemoHandler struct {
	demo  *service.DemoService
	token string
}

// NewDemoHandler requires a non-empty token — config.Load already refuses
// to start the server with DEMO_MODE=true and no DEMO_RESET_TOKEN set, so
// token here is guaranteed non-empty in practice.
func NewDemoHandler(demo *service.DemoService, token string) *DemoHandler {
	return &DemoHandler{demo: demo, token: token}
}

type demoResetResponse struct {
	Reset           bool `json:"reset"`
	Products        int  `json:"products"`
	Sales           int  `json:"sales"`
	MarketingAssets int  `json:"marketing_assets"`
}

// Reset requires an exact, constant-time match of X-Demo-Reset-Token
// against h.token, so a missing or wrong header is indistinguishable from
// each other timing-wise and never executes the reset.
func (h *DemoHandler) Reset(c *fiber.Ctx) error {
	provided := c.Get(DemoResetTokenHeader)
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.token)) != 1 {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or missing demo reset token")
	}

	summary, err := h.demo.Reset(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "demo reset failed")
	}

	return c.JSON(demoResetResponse{
		Reset:           true,
		Products:        summary.Products,
		Sales:           summary.Sales,
		MarketingAssets: summary.MarketingAssets,
	})
}
