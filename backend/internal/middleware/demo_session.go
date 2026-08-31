package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// DemoSessionCookie is the name of the anonymous session cookie used to
// attribute POST /inventory/suggest calls to a visitor for demoquota's
// per-session limit. Demo-only — see CLAUDE.md's "DEMO MODE" section; this
// middleware is only ever registered when config.Config.DemoMode is true.
const DemoSessionCookie = "demo_session_id"

// LocalsDemoSessionID is the fiber.Ctx.Locals key DemoSession stores the
// resolved session UUID under.
const LocalsDemoSessionID = "demo_session_id"

// demoSessionCookieTTL mirrors the 30-day lifetime the rest of this
// project's persistent tokens use — there's no reason for a demo visitor's
// quota identity to expire sooner than that.
const demoSessionCookieTTL = 30 * 24 * time.Hour

// DemoSession assigns (or reuses) a long-lived, httpOnly demo_session_id
// cookie for any visitor hitting a route it guards, and stores the resolved
// UUID in c.Locals(LocalsDemoSessionID) for downstream handlers/middleware
// (see the demo AI quota check ahead of InventorySuggestionHandler.Suggest).
// It never rejects a request — an untrusted or malformed cookie is simply
// replaced with a fresh one.
func DemoSession(secureCookies bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Cookies(DemoSessionCookie))
		if err != nil {
			id = uuid.New()
			c.Cookie(&fiber.Cookie{
				Name:     DemoSessionCookie,
				Value:    id.String(),
				Expires:  time.Now().Add(demoSessionCookieTTL),
				HTTPOnly: true,
				Secure:   secureCookies,
				SameSite: fiber.CookieSameSiteLaxMode,
			})
		}

		c.Locals(LocalsDemoSessionID, id)
		return c.Next()
	}
}
