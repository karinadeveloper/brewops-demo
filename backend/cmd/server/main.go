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
	"github.com/kariaranelly/brew-ops/backend/internal/handler"
	appmiddleware "github.com/kariaranelly/brew-ops/backend/internal/middleware"
	"github.com/kariaranelly/brew-ops/backend/internal/repository"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
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

	registerRoutes(app, cfg, pool)

	return app.Listen(":" + cfg.Port)
}

func registerRoutes(app *fiber.App, cfg *config.Config, pool *pgxpool.Pool) {
	jwtSecret := []byte(cfg.JWTSecret)

	userRepo := repository.NewUserRepository(pool)
	productRepo := repository.NewProductRepository(pool)

	authService := service.NewAuthService(userRepo, jwtSecret, cfg.JWTAccessTokenTTL, cfg.JWTRefreshTokenTTL)
	productService := service.NewProductService(productRepo)

	authHandler := handler.NewAuthHandler(authService, cfg.IsDevelopment())
	productHandler := handler.NewProductHandler(productService)

	v1 := app.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/register", authHandler.Register)

	// Auth applies only to this group and any future protected group
	// (sales, inventory, marketing, reports) — scoped explicitly rather
	// than as a blanket v1.Use(), so it can never accidentally shadow
	// /api/v1/auth/* based on route registration order.
	products := v1.Group("/products", appmiddleware.Auth(jwtSecret))
	products.Get("/", productHandler.List)
	products.Post("/", productHandler.Create)
	products.Get("/trash", productHandler.Trash)
	products.Get("/low-stock", productHandler.LowStock)
	products.Get("/:id", productHandler.Get)
	products.Patch("/:id", productHandler.Update)
	products.Delete("/:id", productHandler.Delete)
	products.Post("/:id/restore", productHandler.Restore)
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
