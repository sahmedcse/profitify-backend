package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds API server configuration loaded from environment variables.
type Config struct {
	DatabaseURL  string
	APIPort      string
	AppEnv       string // "development", "staging", "production"
	PoolMaxConns int
}

// Load reads API server configuration from environment variables.
// It fails fast if required variables are missing.
func Load() (*Config, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL:  dbURL,
		APIPort:      envOrDefault("API_PORT", "8080"),
		AppEnv:       envOrDefault("APP_ENV", "development"),
		PoolMaxConns: intOrDefault("DB_POOL_MAX_CONNS", 4),
	}, nil
}

// FetchTickersConfig holds configuration for the fetch-tickers Lambda.
type FetchTickersConfig struct {
	MassiveAPIKey   string
	SQSQueueURL     string
	TickerLimit     int
	TickerAllowlist []string
}

// LoadFetchTickers reads fetch-tickers Lambda configuration.
func LoadFetchTickers() (*FetchTickersConfig, error) {
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}
	sqsURL, err := required("SQS_QUEUE_URL")
	if err != nil {
		return nil, err
	}

	return &FetchTickersConfig{
		MassiveAPIKey:   apiKey,
		SQSQueueURL:     sqsURL,
		TickerLimit:     intOrDefault("TICKER_LIMIT", 0),
		TickerAllowlist: csvToSlice("TICKER_ALLOWLIST"),
	}, nil
}

// StartPipelineConfig holds configuration for the start-pipeline Lambda.
type StartPipelineConfig struct {
	DatabaseURL  string
	SFNArn       string
	PoolMaxConns int
}

// LoadStartPipeline reads start-pipeline Lambda configuration.
func LoadStartPipeline() (*StartPipelineConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	sfnArn, err := required("SFN_ARN")
	if err != nil {
		return nil, err
	}

	return &StartPipelineConfig{
		DatabaseURL:  dbURL,
		SFNArn:       sfnArn,
		PoolMaxConns: intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// IngestOHLCVConfig holds configuration for the ingest-ohlcv Lambda.
type IngestOHLCVConfig struct {
	DatabaseURL   string
	MassiveAPIKey string
	PoolMaxConns  int
}

// LoadIngestOHLCV reads ingest-ohlcv Lambda configuration.
func LoadIngestOHLCV() (*IngestOHLCVConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}

	return &IngestOHLCVConfig{
		DatabaseURL:   dbURL,
		MassiveAPIKey: apiKey,
		PoolMaxConns:  intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// FetchTechnicalsConfig holds configuration for the fetch-technicals Lambda.
type FetchTechnicalsConfig struct {
	DatabaseURL   string
	MassiveAPIKey string
	PoolMaxConns  int
}

// LoadFetchTechnicals reads fetch-technicals Lambda configuration.
func LoadFetchTechnicals() (*FetchTechnicalsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}

	return &FetchTechnicalsConfig{
		DatabaseURL:   dbURL,
		MassiveAPIKey: apiKey,
		PoolMaxConns:  intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// FetchFundamentalsConfig holds configuration for the fetch-fundamentals Lambda.
type FetchFundamentalsConfig struct {
	DatabaseURL   string
	MassiveAPIKey string
	PoolMaxConns  int
}

// LoadFetchFundamentals reads fetch-fundamentals Lambda configuration.
func LoadFetchFundamentals() (*FetchFundamentalsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}

	return &FetchFundamentalsConfig{
		DatabaseURL:   dbURL,
		MassiveAPIKey: apiKey,
		PoolMaxConns:  intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// EnrichTickerConfig holds configuration for the enrich-ticker Lambda.
type EnrichTickerConfig struct {
	DatabaseURL   string
	MassiveAPIKey string
	PoolMaxConns  int
}

// LoadEnrichTicker reads enrich-ticker Lambda configuration.
func LoadEnrichTicker() (*EnrichTickerConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}

	return &EnrichTickerConfig{
		DatabaseURL:   dbURL,
		MassiveAPIKey: apiKey,
		PoolMaxConns:  intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// ComputeStatsConfig holds configuration for the compute-stats Lambda.
type ComputeStatsConfig struct {
	DatabaseURL   string
	MassiveAPIKey string
	PoolMaxConns  int
}

// LoadComputeStats reads compute-stats Lambda configuration.
func LoadComputeStats() (*ComputeStatsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, err := required("MASSIVE_API_KEY")
	if err != nil {
		return nil, err
	}

	return &ComputeStatsConfig{
		DatabaseURL:   dbURL,
		MassiveAPIKey: apiKey,
		PoolMaxConns:  intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// ClosePipelineConfig holds configuration for the close-pipeline Lambda.
type ClosePipelineConfig struct {
	DatabaseURL  string
	PoolMaxConns int
}

// LoadClosePipeline reads close-pipeline Lambda configuration.
func LoadClosePipeline() (*ClosePipelineConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &ClosePipelineConfig{
		DatabaseURL:  dbURL,
		PoolMaxConns: intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

func required(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: %s is required", key)
	}
	return v, nil
}

func intOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func csvToSlice(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, strings.ToUpper(s))
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
