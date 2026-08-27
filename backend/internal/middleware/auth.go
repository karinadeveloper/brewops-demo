package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/token"
)

// LocalsUserID is the fiber.Ctx.Locals key Auth stores the authenticated
// user's ID under.
const LocalsUserID = "user_id"

// publicPaths never require an access token, regardless of method.
var publicPaths = map[string]bool{
	"/api/v1/auth/login":    true,
	"/api/v1/auth/refresh":  true,
	"/api/v1/auth/register": true,
}

// Auth validates the Authorization: Bearer <access token> header on every
// /api/v1 route except the public auth endpoints (/health sits outside
// /api/v1 entirely, so it never reaches this middleware). On success it
// stores the authenticated user's ID in c.Locals(LocalsUserID).
func Auth(jwtSecret []byte) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if publicPaths[c.Path()] {
			return c.Next()
		}

		header := c.Get(fiber.HeaderAuthorization)
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			return fiber.NewError(fiber.StatusUnauthorized, "missing or malformed authorization header")
		}

		claims, err := token.Parse(jwtSecret, strings.TrimPrefix(header, prefix), token.KindAccess)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals(LocalsUserID, claims.Subject)
		return c.Next()
	}
}
