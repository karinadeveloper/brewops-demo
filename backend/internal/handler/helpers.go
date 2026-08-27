package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/middleware"
)

// currentUserID reads the authenticated user's ID stored by
// middleware.Auth. Handlers behind that middleware can trust it is always
// present and well-formed.
func currentUserID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals(middleware.LocalsUserID).(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, errors.New("missing authenticated user")
	}
	return id, nil
}
