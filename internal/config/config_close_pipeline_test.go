package config

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

func TestLoadClosePipeline(t *testing.T) {
	t.Run("success with default pool", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://localhost/db")

		cfg, err := LoadClosePipeline()
		if err != nil {
			t.Fatalf("LoadClosePipeline() error = %v", err)
		}
		if cfg.DatabaseURL != "postgres://localhost/db" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if cfg.PoolMaxConns != 1 {
			t.Errorf("PoolMaxConns = %d, want default 1", cfg.PoolMaxConns)
		}
	})

	t.Run("pool override", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://localhost/db")
		t.Setenv("DB_POOL_MAX_CONNS", "5")

		cfg, err := LoadClosePipeline()
		if err != nil {
			t.Fatalf("LoadClosePipeline() error = %v", err)
		}
		if cfg.PoolMaxConns != 5 {
			t.Errorf("PoolMaxConns = %d, want 5", cfg.PoolMaxConns)
		}
	})

	t.Run("missing DATABASE_URL", func(t *testing.T) {
		testutil.ClearEnv(t)
		if _, err := LoadClosePipeline(); err == nil {
			t.Fatal("LoadClosePipeline() error = nil, want error")
		}
	})
}
