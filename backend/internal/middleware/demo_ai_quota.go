package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/demoquota"
)

// DemoAIQuota gates POST /inventory/suggest behind demoquota's two-tier
// daily limit. Must run after DemoSession (it reads
// c.Locals(LocalsDemoSessionID)) and before the real handler. Demo-only —
// only ever registered when config.Config.DemoMode is true; the real
// product applies no such limit.
func DemoAIQuota(checker *demoquota.Checker) fiber.Handler {
	return func(c *fiber.Ctx) error {
		sessionID, ok := c.Locals(LocalsDemoSessionID).(uuid.UUID)
		if !ok {
			return fiber.NewError(fiber.StatusInternalServerError, "demo session not initialized")
		}

		result, err := checker.Check(c.Context(), sessionID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "could not evaluate demo AI usage quota")
		}
		if !result.Allowed {
			return fiber.NewError(fiber.StatusTooManyRequests, result.Message)
		}

		return c.Next()
	}
}
