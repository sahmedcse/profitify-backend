package config

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

func TestLoad(t *testing.T) {
	t.Run("defaults applied", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://localhost/db")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.DatabaseURL != "postgres://localhost/db" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.APIPort != "8080" {
			t.Errorf("APIPort = %q, want default 8080", cfg.APIPort)
		}
		if cfg.AppEnv != "development" {
			t.Errorf("AppEnv = %q, want default development", cfg.AppEnv)
		}
		if cfg.PoolMaxConns != 4 {
			t.Errorf("PoolMaxConns = %d, want default 4", cfg.PoolMaxConns)
		}
	})

	t.Run("overrides applied", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://localhost/db")
		t.Setenv("API_PORT", "9999")
		t.Setenv("APP_ENV", "production")
		t.Setenv("DB_POOL_MAX_CONNS", "16")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.APIPort != "9999" || cfg.AppEnv != "production" || cfg.PoolMaxConns != 16 {
			t.Errorf("overrides not applied: %+v", cfg)
		}
	})

	t.Run("missing DATABASE_URL", func(t *testing.T) {
		testutil.ClearEnv(t)
		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want error for missing DATABASE_URL")
		}
	})
}
