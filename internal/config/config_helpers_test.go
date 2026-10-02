package config

import "testing"

// loaderResult is the common projection of every Lambda config loader's
// fields relevant to the cross-cutting tests in this package: the Massive
// API key requirement (config_massive_key_test.go) and DBSecretARN
// (config_db_secret_test.go). Fields that don't apply to a given loader
// (e.g. the Massive key fields for LoadClosePipeline) are left zero.
type loaderResult struct {
	databaseURL            string
	dbSecretARN            string
	massiveAPIKey          string
	massiveAPIKeySecretARN string
	poolMaxConns           int
}

// loader describes one of the eight Lambda config loaders. It is the single
// source of truth shared by config_massive_key_test.go (which tests the
// MASSIVE_API_KEY / MASSIVE_API_KEY_SECRET_ARN requirement) and
// config_db_secret_test.go (which tests DBSecretARN), replacing what used to
// be three separate, near-identical loader tables.
type loader struct {
	name string
	// hasMassiveKey is true for the five loaders that accept MASSIVE_API_KEY
	// / MASSIVE_API_KEY_SECRET_ARN.
	hasMassiveKey bool
	// dbAndKeyShaped is true for the four loaders whose only required env
	// vars are DATABASE_URL and the Massive key. It excludes LoadFetchTickers
	// (also requires SQS_QUEUE_URL; exercised on its own in
	// config_fetch_tickers_test.go) and the three loaders with no Massive key.
	dbAndKeyShaped bool
	// setExtraEnv sets any env vars the loader requires beyond DATABASE_URL
	// and the Massive key (e.g. SQS_QUEUE_URL, SFN_ARN).
	setExtraEnv func(t *testing.T)
	load        func() (loaderResult, error)
}

// setBaseEnv sets DATABASE_URL, the loader's own extra env, and — for the
// five Massive-key loaders — a MASSIVE_API_KEY, so the loader succeeds
// regardless of what a test under config_db_secret_test.go is varying.
func (l loader) setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	l.setExtraEnv(t)
	if l.hasMassiveKey {
		t.Setenv("MASSIVE_API_KEY", "key")
	}
}

func loaders() []loader {
	noExtraEnv := func(*testing.T) {}

	return []loader{
		{
			name:          "LoadFetchTickers",
			hasMassiveKey: true,
			setExtraEnv:   func(t *testing.T) { t.Setenv("SQS_QUEUE_URL", "https://sqs/q") },
			load: func() (loaderResult, error) {
				c, err := LoadFetchTickers()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{
					databaseURL:            c.DatabaseURL,
					dbSecretARN:            c.DBSecretARN,
					massiveAPIKey:          c.MassiveAPIKey,
					massiveAPIKeySecretARN: c.MassiveAPIKeySecretARN,
					poolMaxConns:           c.PoolMaxConns,
				}, nil
			},
		},
		{
			name:        "LoadStartPipeline",
			setExtraEnv: func(t *testing.T) { t.Setenv("SFN_ARN", "arn:aws:states:::sm") },
			load: func() (loaderResult, error) {
				c, err := LoadStartPipeline()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{databaseURL: c.DatabaseURL, dbSecretARN: c.DBSecretARN, poolMaxConns: c.PoolMaxConns}, nil
			},
		},
		{
			name:           "LoadIngestOHLCV",
			hasMassiveKey:  true,
			dbAndKeyShaped: true,
			setExtraEnv:    noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadIngestOHLCV()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{
					databaseURL:            c.DatabaseURL,
					dbSecretARN:            c.DBSecretARN,
					massiveAPIKey:          c.MassiveAPIKey,
					massiveAPIKeySecretARN: c.MassiveAPIKeySecretARN,
					poolMaxConns:           c.PoolMaxConns,
				}, nil
			},
		},
		{
			name:           "LoadFetchTechnicals",
			hasMassiveKey:  true,
			dbAndKeyShaped: true,
			setExtraEnv:    noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadFetchTechnicals()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{
					databaseURL:            c.DatabaseURL,
					dbSecretARN:            c.DBSecretARN,
					massiveAPIKey:          c.MassiveAPIKey,
					massiveAPIKeySecretARN: c.MassiveAPIKeySecretARN,
					poolMaxConns:           c.PoolMaxConns,
				}, nil
			},
		},
		{
			name:           "LoadFetchFundamentals",
			hasMassiveKey:  true,
			dbAndKeyShaped: true,
			setExtraEnv:    noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadFetchFundamentals()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{
					databaseURL:            c.DatabaseURL,
					dbSecretARN:            c.DBSecretARN,
					massiveAPIKey:          c.MassiveAPIKey,
					massiveAPIKeySecretARN: c.MassiveAPIKeySecretARN,
					poolMaxConns:           c.PoolMaxConns,
				}, nil
			},
		},
		{
			name:        "LoadEnrichTicker",
			setExtraEnv: noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadEnrichTicker()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{databaseURL: c.DatabaseURL, dbSecretARN: c.DBSecretARN, poolMaxConns: c.PoolMaxConns}, nil
			},
		},
		{
			name:           "LoadComputeStats",
			hasMassiveKey:  true,
			dbAndKeyShaped: true,
			setExtraEnv:    noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadComputeStats()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{
					databaseURL:            c.DatabaseURL,
					dbSecretARN:            c.DBSecretARN,
					massiveAPIKey:          c.MassiveAPIKey,
					massiveAPIKeySecretARN: c.MassiveAPIKeySecretARN,
					poolMaxConns:           c.PoolMaxConns,
				}, nil
			},
		},
		{
			name:        "LoadClosePipeline",
			setExtraEnv: noExtraEnv,
			load: func() (loaderResult, error) {
				c, err := LoadClosePipeline()
				if err != nil {
					return loaderResult{}, err
				}
				return loaderResult{databaseURL: c.DatabaseURL, dbSecretARN: c.DBSecretARN, poolMaxConns: c.PoolMaxConns}, nil
			},
		},
	}
}

// filterLoaders returns the loaders for which keep returns true, preserving
// order.
func filterLoaders(keep func(loader) bool) []loader {
	all := loaders()
	out := make([]loader, 0, len(all))
	for _, l := range all {
		if keep(l) {
			out = append(out, l)
		}
	}
	return out
}
