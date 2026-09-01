// Command server is the entrypoint for the BrewOps API.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/kariaranelly/brew-ops/backend/config"
	"github.com/kariaranelly/brew-ops/backend/internal/ai"
	"github.com/kariaranelly/brew-ops/backend/internal/demoquota"
	"github.com/kariaranelly/brew-ops/backend/internal/demoseed"
	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/handler"
	appmiddleware "github.com/kariaranelly/brew-ops/backend/internal/middleware"
	"github.com/kariaranelly/brew-ops/backend/internal/repository"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
	"github.com/kariaranelly/brew-ops/backend/internal/storage"
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

	storageClient, err := newStorageClient(ctx, cfg)
	if err != nil {
		return err
	}

	app := fiber.New(fiber.Config{
		// Fiber's default BodyLimit is 4MB, which would silently reject
		// exactly the >15MB uploads CLAUDE.md's image scheme says the
		// backend must accept and optimize rather than reject. This is a
		// technical safety ceiling against unbounded memory use, not a
		// business-rule rejection — no legitimate phone photo gets close
		// to it.
		BodyLimit: 50 * 1024 * 1024,
	})
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowedOrigins,
		// AllowCredentials is required for the browser to send the
		// demo_session_id cookie (see appmiddleware.DemoSession) back on
		// cross-origin requests from the Vite dev server / deployed
		// frontend origin. Safe with a non-wildcard AllowOrigins, which is
		// already the case here.
		AllowCredentials: true,
	}))

	// General rate-limit tier: applies to all of /api/v1, generous enough to
	// never bother a single admin, strict enough to catch a runaway
	// loop/bot before it burns through Cloud Run's free-tier quota — see
	// CLAUDE.md's "Rate limiting" section.
	app.Use("/api/v1", limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "too many requests, slow down")
		},
	}))

	if cfg.StorageBackend == "local" {
		app.Static("/local-storage", localStorageDir)
	}

	app.Get("/health", healthHandler(pool))

	registerRoutes(app, cfg, pool, storageClient)

	return app.Listen(":" + cfg.Port)
}

// localStorageDir is where LocalDiskStorageClient writes uploaded files —
// see CLAUDE.md's "no active GCP billing account yet" note. Gitignored.
const localStorageDir = "./local-storage"

// newStorageClient selects the image storage backend from
// cfg.StorageBackend. "local" (default) needs no GCP setup at all, which is
// the point — see storage.LocalDiskStorageClient's doc comment.
func newStorageClient(ctx context.Context, cfg *config.Config) (domain.StorageClient, error) {
	switch cfg.StorageBackend {
	case "gcs":
		return storage.NewGCSStorageClient(ctx, cfg.GCPStorageBucket)
	case "local", "":
		return storage.NewLocalDiskStorageClient(localStorageDir, cfg.PublicBaseURL)
	default:
		return nil, fmt.Errorf("config: unknown STORAGE_BACKEND %q (want \"local\" or \"gcs\")", cfg.StorageBackend)
	}
}

func registerRoutes(app *fiber.App, cfg *config.Config, pool *pgxpool.Pool, storageClient domain.StorageClient) {
	jwtSecret := []byte(cfg.JWTSecret)

	userRepo := repository.NewUserRepository(pool)
	productRepo := repository.NewProductRepository(pool)
	inventoryRepo := repository.NewInventoryRepository(pool)
	saleRepo := repository.NewSaleRepository(pool)
	marketingAssetRepo := repository.NewMarketingAssetRepository(pool)
	reportRepo := repository.NewReportRepository(pool)

	authService := service.NewAuthService(userRepo, jwtSecret, cfg.JWTAccessTokenTTL, cfg.JWTRefreshTokenTTL)
	productService := service.NewProductService(productRepo)
	inventoryService := service.NewInventoryService(inventoryRepo)
	saleService := service.NewSaleService(saleRepo)
	imageService := service.NewImageService(storageClient)
	marketingAssetService := service.NewMarketingAssetService(marketingAssetRepo, imageService)
	openAIClient := ai.NewOpenAIClient(cfg.OpenAIAPIKey)
	suggestionService := service.NewInventorySuggestionService(openAIClient, productRepo)
	reportService := service.NewReportService(reportRepo)

	authHandler := handler.NewAuthHandler(authService, cfg.IsDevelopment())
	productHandler := handler.NewProductHandler(productService)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)
	saleHandler := handler.NewSaleHandler(saleService)
	uploadHandler := handler.NewUploadHandler(imageService)
	marketingAssetHandler := handler.NewMarketingAssetHandler(marketingAssetService)
	suggestionHandler := handler.NewInventorySuggestionHandler(suggestionService)
	reportHandler := handler.NewReportHandler(reportService)

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

	movements := v1.Group("/inventory/movements", appmiddleware.Auth(jwtSecret))
	movements.Get("/", inventoryHandler.List)
	movements.Post("/", inventoryHandler.Create)
	movements.Get("/trash", inventoryHandler.Trash)
	movements.Delete("/:id", inventoryHandler.Delete)
	movements.Post("/:id/restore", inventoryHandler.Restore)

	// AI rate-limit tier: a much stricter, independent limit than the
	// general tier, because every call here costs real money via the
	// OpenAI API regardless of server load — see CLAUDE.md's "Rate
	// limiting" section. 5/min (not the real product's 10/min) — this repo
	// is always the public demo, so a tighter per-IP ceiling is a
	// permanent, unconditional defense-in-depth layer here, not something
	// gated by DEMO_MODE — see CLAUDE.md's "DEMO MODE" section.
	suggestMiddlewares := []fiber.Handler{appmiddleware.Auth(jwtSecret), limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		LimitReached: func(c *fiber.Ctx) error {
			return fiber.NewError(fiber.StatusTooManyRequests, "AI suggestion rate limit exceeded, try again in a minute")
		},
	})}

	// Demo-only two-tier AI usage quota (per-session + global daily caps) —
	// only wired up when DEMO_MODE=true. The real product applies no such
	// quota; a single admin's normal usage is the only limit there. See
	// CLAUDE.md's "DEMO MODE" section.
	if cfg.DemoMode {
		quotaChecker := demoquota.NewChecker(demoquota.NewPostgresStore(pool))
		suggestMiddlewares = append(suggestMiddlewares,
			appmiddleware.DemoSession(!cfg.IsDevelopment()),
			appmiddleware.DemoAIQuota(quotaChecker),
		)
	}

	suggest := v1.Group("/inventory", suggestMiddlewares...)
	suggest.Post("/suggest", suggestionHandler.Suggest)

	sales := v1.Group("/sales", appmiddleware.Auth(jwtSecret))
	sales.Get("/", saleHandler.List)
	sales.Post("/", saleHandler.Create)
	sales.Get("/trash", saleHandler.Trash)
	sales.Get("/:id", saleHandler.Get)
	sales.Delete("/:id", saleHandler.Delete)
	sales.Post("/:id/restore", saleHandler.Restore)

	uploads := v1.Group("/uploads", appmiddleware.Auth(jwtSecret))
	uploads.Post("/image", uploadHandler.Upload)

	marketingAssets := v1.Group("/marketing/assets", appmiddleware.Auth(jwtSecret))
	marketingAssets.Get("/", marketingAssetHandler.List)
	marketingAssets.Post("/", marketingAssetHandler.Create)
	marketingAssets.Get("/trash", marketingAssetHandler.Trash)
	marketingAssets.Delete("/:id", marketingAssetHandler.Delete)
	marketingAssets.Post("/:id/restore", marketingAssetHandler.Restore)

	reports := v1.Group("/reports", appmiddleware.Auth(jwtSecret))
	reports.Get("/revenue", reportHandler.Revenue)
	reports.Get("/top-products", reportHandler.TopProducts)
	reports.Get("/inventory-value", reportHandler.InventoryValue)

	// Demo-only reset endpoint — deliberately not registered at all unless
	// DEMO_MODE=true, so a request against a non-demo deployment gets
	// Fiber's plain unmatched-route 404 instead of a handler that reveals
	// the route exists at all. See CLAUDE.md's "DEMO MODE" section.
	if cfg.DemoMode {
		demoService := service.NewDemoService(pool, demoseed.Options{
			AdminEmail:      cfg.DemoAdminEmail,
			AdminPassword:   cfg.DemoAdminPassword,
			SeedAssetsDir:   cfg.SeedAssetsDir,
			LocalStorageDir: localStorageDir,
			PublicBaseURL:   cfg.PublicBaseURL,
		})
		demoHandler := handler.NewDemoHandler(demoService, cfg.DemoResetToken)
		v1.Post("/admin/demo-reset", demoHandler.Reset)
	}
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
