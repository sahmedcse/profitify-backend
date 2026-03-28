package config

import (
	"fmt"
	"os"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	APIPort     string
	AppEnv      string // "development", "staging", "production"
}

// Load reads configuration from environment variables.
// It fails fast if required variables are missing.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	return &Config{
		DatabaseURL: dbURL,
		APIPort:     port,
		AppEnv:      appEnv,
	}, nil
}
