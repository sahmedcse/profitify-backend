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
	MassiveAPIKey          string
	MassiveAPIKeySecretARN string
	SQSQueueURL            string
	TickerLimit            int
	TickerAllowlist        []string
	DatabaseURL            string
	DBSecretARN            string
	PoolMaxConns           int
}

// LoadFetchTickers reads fetch-tickers Lambda configuration.
func LoadFetchTickers() (*FetchTickersConfig, error) {
	apiKey, apiKeySecretARN, err := requiredMassiveAPIKey()
	if err != nil {
		return nil, err
	}
	sqsURL, err := required("SQS_QUEUE_URL")
	if err != nil {
		return nil, err
	}
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &FetchTickersConfig{
		MassiveAPIKey:          apiKey,
		MassiveAPIKeySecretARN: apiKeySecretARN,
		SQSQueueURL:            sqsURL,
		TickerLimit:            intOrDefault("TICKER_LIMIT", 0),
		TickerAllowlist:        csvToSlice("TICKER_ALLOWLIST"),
		DatabaseURL:            dbURL,
		DBSecretARN:            os.Getenv("DB_SECRET_ARN"),
		PoolMaxConns:           intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// StartPipelineConfig holds configuration for the start-pipeline Lambda.
type StartPipelineConfig struct {
	DatabaseURL  string
	DBSecretARN  string
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
		DBSecretARN:  os.Getenv("DB_SECRET_ARN"),
		SFNArn:       sfnArn,
		PoolMaxConns: intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// IngestOHLCVConfig holds configuration for the ingest-ohlcv Lambda.
type IngestOHLCVConfig struct {
	DatabaseURL            string
	DBSecretARN            string
	MassiveAPIKey          string
	MassiveAPIKeySecretARN string
	PoolMaxConns           int
}

// LoadIngestOHLCV reads ingest-ohlcv Lambda configuration.
func LoadIngestOHLCV() (*IngestOHLCVConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, apiKeySecretARN, err := requiredMassiveAPIKey()
	if err != nil {
		return nil, err
	}

	return &IngestOHLCVConfig{
		DatabaseURL:            dbURL,
		DBSecretARN:            os.Getenv("DB_SECRET_ARN"),
		MassiveAPIKey:          apiKey,
		MassiveAPIKeySecretARN: apiKeySecretARN,
		PoolMaxConns:           intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// FetchTechnicalsConfig holds configuration for the fetch-technicals Lambda.
type FetchTechnicalsConfig struct {
	DatabaseURL            string
	DBSecretARN            string
	MassiveAPIKey          string
	MassiveAPIKeySecretARN string
	PoolMaxConns           int
}

// LoadFetchTechnicals reads fetch-technicals Lambda configuration.
func LoadFetchTechnicals() (*FetchTechnicalsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, apiKeySecretARN, err := requiredMassiveAPIKey()
	if err != nil {
		return nil, err
	}

	return &FetchTechnicalsConfig{
		DatabaseURL:            dbURL,
		DBSecretARN:            os.Getenv("DB_SECRET_ARN"),
		MassiveAPIKey:          apiKey,
		MassiveAPIKeySecretARN: apiKeySecretARN,
		PoolMaxConns:           intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// FetchFundamentalsConfig holds configuration for the fetch-fundamentals Lambda.
type FetchFundamentalsConfig struct {
	DatabaseURL            string
	DBSecretARN            string
	MassiveAPIKey          string
	MassiveAPIKeySecretARN string
	PoolMaxConns           int
}

// LoadFetchFundamentals reads fetch-fundamentals Lambda configuration.
func LoadFetchFundamentals() (*FetchFundamentalsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, apiKeySecretARN, err := requiredMassiveAPIKey()
	if err != nil {
		return nil, err
	}

	return &FetchFundamentalsConfig{
		DatabaseURL:            dbURL,
		DBSecretARN:            os.Getenv("DB_SECRET_ARN"),
		MassiveAPIKey:          apiKey,
		MassiveAPIKeySecretARN: apiKeySecretARN,
		PoolMaxConns:           intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// EnrichTickerConfig holds configuration for the enrich-ticker Lambda. It
// never builds a Massive client, so it has no Massive API key field.
type EnrichTickerConfig struct {
	DatabaseURL  string
	DBSecretARN  string
	PoolMaxConns int
}

// LoadEnrichTicker reads enrich-ticker Lambda configuration.
func LoadEnrichTicker() (*EnrichTickerConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	return &EnrichTickerConfig{
		DatabaseURL:  dbURL,
		DBSecretARN:  os.Getenv("DB_SECRET_ARN"),
		PoolMaxConns: intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// ComputeStatsConfig holds configuration for the compute-stats Lambda.
type ComputeStatsConfig struct {
	DatabaseURL            string
	DBSecretARN            string
	MassiveAPIKey          string
	MassiveAPIKeySecretARN string
	PoolMaxConns           int
}

// LoadComputeStats reads compute-stats Lambda configuration.
func LoadComputeStats() (*ComputeStatsConfig, error) {
	dbURL, err := required("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	apiKey, apiKeySecretARN, err := requiredMassiveAPIKey()
	if err != nil {
		return nil, err
	}

	return &ComputeStatsConfig{
		DatabaseURL:            dbURL,
		DBSecretARN:            os.Getenv("DB_SECRET_ARN"),
		MassiveAPIKey:          apiKey,
		MassiveAPIKeySecretARN: apiKeySecretARN,
		PoolMaxConns:           intOrDefault("DB_POOL_MAX_CONNS", 1),
	}, nil
}

// ClosePipelineConfig holds configuration for the close-pipeline Lambda.
type ClosePipelineConfig struct {
	DatabaseURL  string
	DBSecretARN  string
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
		DBSecretARN:  os.Getenv("DB_SECRET_ARN"),
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

// requiredMassiveAPIKey reads MASSIVE_API_KEY and MASSIVE_API_KEY_SECRET_ARN,
// requiring at least one to be set. It is shared by every Lambda config
// loader that builds a Massive client (fetch-tickers, ingest-ohlcv,
// fetch-technicals, fetch-fundamentals, compute-stats). Either value may be
// returned empty; the caller (via internal/lambda.MassiveAPIKey) prefers the
// secret when both are set.
func requiredMassiveAPIKey() (apiKey, secretARN string, err error) {
	apiKey = os.Getenv("MASSIVE_API_KEY")
	secretARN = os.Getenv("MASSIVE_API_KEY_SECRET_ARN")
	if apiKey == "" && secretARN == "" {
		return "", "", fmt.Errorf("config: MASSIVE_API_KEY or MASSIVE_API_KEY_SECRET_ARN is required")
	}
	return apiKey, secretARN, nil
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
	return NormalizeSymbols(strings.Split(v, ","))
}

// NormalizeSymbols trims whitespace, upper-cases, drops blank entries, and
// removes duplicates while preserving first-seen order. It is the single
// normalizer shared by env-var allowlists (via csvToSlice) and event-supplied
// ticker lists. Returns nil when every entry is blank or symbols is empty.
func NormalizeSymbols(symbols []string) []string {
	seen := make(map[string]struct{}, len(symbols))
	result := make([]string, 0, len(symbols))
	for _, s := range symbols {
		trimmed := strings.ToUpper(strings.TrimSpace(s))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
