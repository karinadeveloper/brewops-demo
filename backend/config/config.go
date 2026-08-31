// Package config loads and validates application configuration from
// environment variables. It fails explicitly (returning an error) when a
// required variable is missing, rather than crashing with log.Fatal outside
// of main().
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the BrewOps backend.
type Config struct {
	// Server
	Port        string
	Environment string

	// Database
	DatabaseURL string

	// Auth
	JWTSecret          string
	JWTAccessTokenTTL  time.Duration
	JWTRefreshTokenTTL time.Duration

	// AI feature
	OpenAIAPIKey string

	// GCP Cloud Storage
	GCPProjectID         string
	GCPStorageBucket     string
	GoogleAppCredentials string

	// Image storage backend: "local" (default, ./local-storage/ + a static
	// route) or "gcs" (GCP Cloud Storage) — see storage.StorageClient.
	StorageBackend string
	// Base URL LocalDiskStorageClient prefixes onto the filename to build a
	// publicly-fetchable URL, mirroring GCS's fully-qualified URLs. Unused
	// when StorageBackend is "gcs".
	PublicBaseURL string

	// CORS
	CORSAllowedOrigins string

	// Demo mode — see CLAUDE.md's "DEMO MODE" section. Everything gated by
	// this flag (the reset endpoint, the AI usage quotas) is a demo-only
	// concern that does not exist in the real BrewOps product; it defaults
	// to false (fail closed) so a misconfigured deployment never accidentally
	// exposes demo-only surface area.
	DemoMode bool
	// Credentials for the single demo admin account. Read here (never
	// hardcoded) so both cmd/seed and the reset endpoint upsert the exact
	// same account. Not required unless the seed command or the reset
	// endpoint is actually invoked.
	DemoAdminEmail    string
	DemoAdminPassword string
	// Shared secret POST /api/v1/admin/demo-reset compares against the
	// X-Demo-Reset-Token header. Required whenever DemoMode is true — see
	// handler.DemoHandler.
	DemoResetToken string
}

// Load reads configuration from environment variables and returns an error
// if any required variable is missing or malformed.
func Load() (*Config, error) {
	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		// Defaults to "production" (fail closed): APP_ENV must be set to
		// "development" explicitly to expose POST /api/v1/auth/register.
		Environment:          getEnv("APP_ENV", "production"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		GCPProjectID:         os.Getenv("GCP_PROJECT_ID"),
		GCPStorageBucket:     os.Getenv("GCP_STORAGE_BUCKET"),
		GoogleAppCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		StorageBackend:       getEnv("STORAGE_BACKEND", "local"),
		PublicBaseURL:        getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),

		// Defaults to false (fail closed): DEMO_MODE must be set to "true"
		// explicitly to expose the demo-reset endpoint and the AI usage
		// quotas. Any other value (including unset) keeps this false.
		DemoMode:          getEnv("DEMO_MODE", "false") == "true",
		DemoAdminEmail:    os.Getenv("DEMO_ADMIN_EMAIL"),
		DemoAdminPassword: os.Getenv("DEMO_ADMIN_PASSWORD"),
		DemoResetToken:    os.Getenv("DEMO_RESET_TOKEN"),
	}

	if err := requireAll(map[string]string{
		"DATABASE_URL": cfg.DatabaseURL,
		"JWT_SECRET":   cfg.JWTSecret,
	}); err != nil {
		return nil, err
	}

	// 8h access / 30d refresh — see the "JWT token lifetimes" note in
	// CLAUDE.md's tech stack table for why this differs from the
	// 15min/7days industry default.
	accessTTL, err := parseDuration("JWT_ACCESS_TOKEN_TTL", "8h")
	if err != nil {
		return nil, err
	}
	cfg.JWTAccessTokenTTL = accessTTL

	refreshTTL, err := parseDuration("JWT_REFRESH_TOKEN_TTL", "720h")
	if err != nil {
		return nil, err
	}
	cfg.JWTRefreshTokenTTL = refreshTTL

	// Fail fast at startup rather than silently rejecting every reset
	// request forever: a demo deployment with DEMO_MODE=true but no reset
	// token configured is a misconfiguration, not a valid "reset disabled"
	// state (that's what DEMO_MODE=false is for).
	if cfg.DemoMode && cfg.DemoResetToken == "" {
		return nil, fmt.Errorf("config: DEMO_RESET_TOKEN is required when DEMO_MODE=true")
	}

	return cfg, nil
}

// IsDevelopment reports whether the server is running in development mode,
// the only mode where POST /api/v1/auth/register is exposed.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func requireAll(required map[string]string) error {
	for key, value := range required {
		if value == "" {
			return fmt.Errorf("config: missing required environment variable %q", key)
		}
	}
	return nil
}

func parseDuration(key, fallback string) (time.Duration, error) {
	raw := getEnv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid duration for %q: %w", key, err)
	}
	return d, nil
}
