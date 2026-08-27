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
	Port string

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

	// CORS
	CORSAllowedOrigins string
}

// Load reads configuration from environment variables and returns an error
// if any required variable is missing or malformed.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		GCPProjectID:         os.Getenv("GCP_PROJECT_ID"),
		GCPStorageBucket:     os.Getenv("GCP_STORAGE_BUCKET"),
		GoogleAppCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		CORSAllowedOrigins:   getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
	}

	if err := requireAll(map[string]string{
		"DATABASE_URL": cfg.DatabaseURL,
		"JWT_SECRET":   cfg.JWTSecret,
	}); err != nil {
		return nil, err
	}

	accessTTL, err := parseDuration("JWT_ACCESS_TOKEN_TTL", "15m")
	if err != nil {
		return nil, err
	}
	cfg.JWTAccessTokenTTL = accessTTL

	refreshTTL, err := parseDuration("JWT_REFRESH_TOKEN_TTL", "168h")
	if err != nil {
		return nil, err
	}
	cfg.JWTRefreshTokenTTL = refreshTTL

	return cfg, nil
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
