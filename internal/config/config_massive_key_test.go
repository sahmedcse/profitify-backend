package config

import (
	"strings"
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

// dbAndKeyLoaders returns the four loaders (see loaders() in
// config_helpers_test.go) whose only required env vars are DATABASE_URL and
// the Massive key: it excludes LoadFetchTickers (also requires
// SQS_QUEUE_URL) and the three loaders with no Massive key.
func dbAndKeyLoaders() []loader {
	return filterLoaders(func(l loader) bool { return l.dbAndKeyShaped })
}

func TestDBAndKeyLoaders_Success(t *testing.T) {
	for _, l := range dbAndKeyLoaders() {
		t.Run(l.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost/db")
			t.Setenv("MASSIVE_API_KEY", "key")

			res, err := l.load()
			if err != nil {
				t.Fatalf("%s() error = %v", l.name, err)
			}
			if res.databaseURL != "postgres://localhost/db" {
				t.Errorf("DatabaseURL = %q", res.databaseURL)
			}
			if res.massiveAPIKey != "key" {
				t.Errorf("MassiveAPIKey = %q", res.massiveAPIKey)
			}
			if res.poolMaxConns != 1 {
				t.Errorf("PoolMaxConns = %d, want default 1", res.poolMaxConns)
			}
		})
	}
}

func TestDBAndKeyLoaders_PoolOverride(t *testing.T) {
	for _, l := range dbAndKeyLoaders() {
		t.Run(l.name, func(t *testing.T) {
			testutil.ClearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost/db")
			t.Setenv("MASSIVE_API_KEY", "key")
			t.Setenv("DB_POOL_MAX_CONNS", "9")

			res, err := l.load()
			if err != nil {
				t.Fatalf("%s() error = %v", l.name, err)
			}
			if res.poolMaxConns != 9 {
				t.Errorf("PoolMaxConns = %d, want 9", res.poolMaxConns)
			}
		})
	}
}

func TestDBAndKeyLoaders_MissingVars(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "no database url", env: map[string]string{"MASSIVE_API_KEY": "key"}},
		{name: "no api key", env: map[string]string{"DATABASE_URL": "postgres://localhost/db"}},
		{name: "neither", env: map[string]string{}},
	}

	for _, l := range dbAndKeyLoaders() {
		for _, c := range cases {
			t.Run(l.name+"/"+c.name, func(t *testing.T) {
				testutil.ClearEnv(t)
				for k, v := range c.env {
					t.Setenv(k, v)
				}
				if _, err := l.load(); err == nil {
					t.Errorf("%s() error = nil, want error", l.name)
				}
			})
		}
	}
}

// TestKeyLoaders_MassiveAPIKeyOrSecretARN covers the five loaders (see
// loaders() in config_helpers_test.go) that require MASSIVE_API_KEY or
// MASSIVE_API_KEY_SECRET_ARN.
func TestKeyLoaders_MassiveAPIKeyOrSecretARN(t *testing.T) {
	const secretARN = "arn:aws:secretsmanager:us-east-1:1:secret:massive"

	keyLoaders := filterLoaders(func(l loader) bool { return l.hasMassiveKey })

	for _, l := range keyLoaders {
		t.Run(l.name, func(t *testing.T) {
			t.Run("secret ARN alone succeeds", func(t *testing.T) {
				testutil.ClearEnv(t)
				t.Setenv("DATABASE_URL", "postgres://localhost/db")
				l.setExtraEnv(t)
				t.Setenv("MASSIVE_API_KEY_SECRET_ARN", secretARN)

				res, err := l.load()
				if err != nil {
					t.Fatalf("%s() error = %v", l.name, err)
				}
				if res.massiveAPIKeySecretARN != secretARN {
					t.Errorf("MassiveAPIKeySecretARN = %q, want %q", res.massiveAPIKeySecretARN, secretARN)
				}
				if res.massiveAPIKey != "" {
					t.Errorf("MassiveAPIKey = %q, want empty when only the secret ARN is set", res.massiveAPIKey)
				}
			})

			t.Run("env key alone succeeds", func(t *testing.T) {
				testutil.ClearEnv(t)
				t.Setenv("DATABASE_URL", "postgres://localhost/db")
				l.setExtraEnv(t)
				t.Setenv("MASSIVE_API_KEY", "key")

				res, err := l.load()
				if err != nil {
					t.Fatalf("%s() error = %v", l.name, err)
				}
				if res.massiveAPIKey != "key" {
					t.Errorf("MassiveAPIKey = %q, want key", res.massiveAPIKey)
				}
				if res.massiveAPIKeySecretARN != "" {
					t.Errorf("MassiveAPIKeySecretARN = %q, want empty when only the env var is set", res.massiveAPIKeySecretARN)
				}
			})

			t.Run("neither gives the or error", func(t *testing.T) {
				testutil.ClearEnv(t)
				t.Setenv("DATABASE_URL", "postgres://localhost/db")
				l.setExtraEnv(t)

				_, err := l.load()
				if err == nil {
					t.Fatalf("%s() error = nil, want error when neither is set", l.name)
				}
				if !strings.Contains(err.Error(), "MASSIVE_API_KEY or MASSIVE_API_KEY_SECRET_ARN is required") {
					t.Errorf("error = %q, want the shared 'or' message", err.Error())
				}
			})
		})
	}
}

func TestLoadEnrichTicker_DoesNotRequireMassiveAPIKey(t *testing.T) {
	testutil.ClearEnv(t)
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	// Deliberately left unset: enrich-ticker never builds a Massive client,
	// so neither MASSIVE_API_KEY nor its secret ARN variant is required.

	cfg, err := LoadEnrichTicker()
	if err != nil {
		t.Fatalf("LoadEnrichTicker() error = %v, want success without a Massive key", err)
	}
	if cfg.DatabaseURL != "postgres://localhost/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.PoolMaxConns != 1 {
		t.Errorf("PoolMaxConns = %d, want default 1", cfg.PoolMaxConns)
	}
}

func TestLoadEnrichTicker_MissingDatabaseURL(t *testing.T) {
	testutil.ClearEnv(t)
	if _, err := LoadEnrichTicker(); err == nil {
		t.Fatal("LoadEnrichTicker() error = nil, want error for missing DATABASE_URL")
	}
}
