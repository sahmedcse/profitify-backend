package config

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

func TestLoadStartPipeline(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("DATABASE_URL", "postgres://localhost/db")
		t.Setenv("SFN_ARN", "arn:aws:states:::sm")

		cfg, err := LoadStartPipeline()
		if err != nil {
			t.Fatalf("LoadStartPipeline() error = %v", err)
		}
		if cfg.SFNArn != "arn:aws:states:::sm" {
			t.Errorf("SFNArn = %q", cfg.SFNArn)
		}
		if cfg.PoolMaxConns != 1 {
			t.Errorf("PoolMaxConns = %d, want default 1", cfg.PoolMaxConns)
		}
	})

	t.Run("missing required vars", func(t *testing.T) {
		cases := []struct {
			name string
			env  map[string]string
		}{
			{name: "no database url", env: map[string]string{"SFN_ARN": "arn"}},
			{name: "no sfn arn", env: map[string]string{"DATABASE_URL": "postgres://localhost/db"}},
			{name: "neither", env: map[string]string{}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				testutil.ClearEnv(t)
				for k, v := range c.env {
					t.Setenv(k, v)
				}
				if _, err := LoadStartPipeline(); err == nil {
					t.Error("LoadStartPipeline() error = nil, want error")
				}
			})
		}
	})
}
