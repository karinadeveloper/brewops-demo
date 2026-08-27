// Command server is the entrypoint for the BrewOps API.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/kariaranelly/brew-ops/backend/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// godotenv only loads .env for local development; in Cloud Run the real
	// environment variables are injected and this is a silent no-op.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	app := fiber.New()
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
	}))

	app.Get("/health", healthHandler(pool))

	// Route registration under this group begins in Session 2.
	app.Group("/api/v1")

	return app.Listen(":" + cfg.Port)
}

func healthHandler(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":   "error",
				"database": "unreachable",
			})
		}

		return c.JSON(fiber.Map{
			"status":   "ok",
			"database": "connected",
		})
	}
}
