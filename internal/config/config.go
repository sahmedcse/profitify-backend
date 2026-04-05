package config

import (
	"fmt"
	"os"
)

// Config holds API server configuration loaded from environment variables.
type Config struct {
	DatabaseURL string
	APIPort     string
	AppEnv      string // "development", "staging", "production"
}

// Load reads API server configuration from environment variables.
// It fails fast if required variables are missing.
func Load() (*Config, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL: dbURL,
		APIPort:     envOrDefault("API_PORT", "8080"),
		AppEnv:      envOrDefault("APP_ENV", "development"),
	}, nil
}

// FetchTickersConfig holds configuration for the fetch-tickers Lambda.
type FetchTickersConfig struct {
	DatabaseURL   string
	PolygonAPIKey string
	SQSQueueURL   string
}

// LoadFetchTickers reads fetch-tickers Lambda configuration.
func LoadFetchTickers() (*FetchTickersConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("POLYGON_API_KEY")
	if err != nil {
		return nil, err
	}
	sqsURL, err := required("SQS_QUEUE_URL")
	if err != nil {
		return nil, err
	}

	return &FetchTickersConfig{
		DatabaseURL:   dbURL,
		PolygonAPIKey: apiKey,
		SQSQueueURL:   sqsURL,
	}, nil
}

// StartPipelineConfig holds configuration for the start-pipeline Lambda.
type StartPipelineConfig struct {
	SFNArn string
}

// LoadStartPipeline reads start-pipeline Lambda configuration.
func LoadStartPipeline() (*StartPipelineConfig, error) {
	sfnArn, err := required("SFN_ARN")
	if err != nil {
		return nil, err
	}

	return &StartPipelineConfig{
		SFNArn: sfnArn,
	}, nil
}

// IngestOHLCVConfig holds configuration for the ingest-ohlcv Lambda.
type IngestOHLCVConfig struct {
	DatabaseURL   string
	PolygonAPIKey string
}

// LoadIngestOHLCV reads ingest-ohlcv Lambda configuration.
func LoadIngestOHLCV() (*IngestOHLCVConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("POLYGON_API_KEY")
	if err != nil {
		return nil, err
	}

	return &IngestOHLCVConfig{
		DatabaseURL:   dbURL,
		PolygonAPIKey: apiKey,
	}, nil
}

// EnrichTickerConfig holds configuration for the enrich-ticker Lambda.
type EnrichTickerConfig struct {
	DatabaseURL   string
	PolygonAPIKey string
}

// LoadEnrichTicker reads enrich-ticker Lambda configuration.
func LoadEnrichTicker() (*EnrichTickerConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("POLYGON_API_KEY")
	if err != nil {
		return nil, err
	}

	return &EnrichTickerConfig{
		DatabaseURL:   dbURL,
		PolygonAPIKey: apiKey,
	}, nil
}

// ComputeStatsConfig holds configuration for the compute-stats Lambda.
type ComputeStatsConfig struct {
	DatabaseURL string
}

// LoadComputeStats reads compute-stats Lambda configuration.
func LoadComputeStats() (*ComputeStatsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &ComputeStatsConfig{
		DatabaseURL: dbURL,
	}, nil
}

func required(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}
	return v, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
