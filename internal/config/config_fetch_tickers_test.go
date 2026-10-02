package config

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

func TestLoadFetchTickers(t *testing.T) {
	t.Run("success with defaults", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("MASSIVE_API_KEY", "key")
		t.Setenv("SQS_QUEUE_URL", "https://sqs/q")
		t.Setenv("DATABASE_URL", "postgres://localhost/db")

		cfg, err := LoadFetchTickers()
		if err != nil {
			t.Fatalf("LoadFetchTickers() error = %v", err)
		}
		if cfg.MassiveAPIKey != "key" || cfg.SQSQueueURL != "https://sqs/q" {
			t.Errorf("unexpected config: %+v", cfg)
		}
		if cfg.TickerLimit != 0 {
			t.Errorf("TickerLimit = %d, want default 0", cfg.TickerLimit)
		}
		if cfg.TickerAllowlist != nil {
			t.Errorf("TickerAllowlist = %v, want nil", cfg.TickerAllowlist)
		}
		if cfg.DatabaseURL != "postgres://localhost/db" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
	})

	t.Run("allowlist and limit parsed", func(t *testing.T) {
		testutil.ClearEnv(t)
		t.Setenv("MASSIVE_API_KEY", "key")
		t.Setenv("SQS_QUEUE_URL", "https://sqs/q")
		t.Setenv("DATABASE_URL", "postgres://localhost/db")
		t.Setenv("TICKER_LIMIT", "50")
		t.Setenv("TICKER_ALLOWLIST", "aapl, msft")

		cfg, err := LoadFetchTickers()
		if err != nil {
			t.Fatalf("LoadFetchTickers() error = %v", err)
		}
		if cfg.TickerLimit != 50 {
			t.Errorf("TickerLimit = %d, want 50", cfg.TickerLimit)
		}
		want := []string{"AAPL", "MSFT"}
		if len(cfg.TickerAllowlist) != len(want) {
			t.Fatalf("TickerAllowlist = %v, want %v", cfg.TickerAllowlist, want)
		}
		for i := range want {
			if cfg.TickerAllowlist[i] != want[i] {
				t.Errorf("TickerAllowlist[%d] = %q, want %q", i, cfg.TickerAllowlist[i], want[i])
			}
		}
	})

	t.Run("missing required vars", func(t *testing.T) {
		cases := []struct {
			name string
			env  map[string]string
		}{
			{name: "no api key", env: map[string]string{"SQS_QUEUE_URL": "https://sqs/q", "DATABASE_URL": "postgres://localhost/db"}},
			{name: "no queue url", env: map[string]string{"MASSIVE_API_KEY": "key", "DATABASE_URL": "postgres://localhost/db"}},
			{name: "no database url", env: map[string]string{"MASSIVE_API_KEY": "key", "SQS_QUEUE_URL": "https://sqs/q"}},
			{name: "neither", env: map[string]string{}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				testutil.ClearEnv(t)
				for k, v := range c.env {
					t.Setenv(k, v)
				}
				if _, err := LoadFetchTickers(); err == nil {
					t.Error("LoadFetchTickers() error = nil, want error")
				}
			})
		}
	})
}

func TestLoadFetchTickers_RequiresDatabaseURL(t *testing.T) {
	testutil.ClearEnv(t)
	t.Setenv("MASSIVE_API_KEY", "key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs/q")

	if _, err := LoadFetchTickers(); err == nil {
		t.Fatal("LoadFetchTickers() error = nil, want error for missing DATABASE_URL")
	}
}

func TestLoadFetchTickers_PoolMaxConnsDefault(t *testing.T) {
	testutil.ClearEnv(t)
	t.Setenv("MASSIVE_API_KEY", "key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs/q")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	cfg, err := LoadFetchTickers()
	if err != nil {
		t.Fatalf("LoadFetchTickers() error = %v", err)
	}
	if cfg.PoolMaxConns != 1 {
		t.Errorf("PoolMaxConns = %d, want default 1", cfg.PoolMaxConns)
	}
}
