package config

import (
	"strings"
	"testing"
)

func TestCsvToSlice(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{name: "empty", env: "", want: nil},
		{name: "single", env: "AAPL", want: []string{"AAPL"}},
		{name: "multiple", env: "AAPL,MSFT,GOOG", want: []string{"AAPL", "MSFT", "GOOG"}},
		{name: "with spaces", env: " AAPL , MSFT , GOOG ", want: []string{"AAPL", "MSFT", "GOOG"}},
		{name: "lowercase to upper", env: "aapl,msft", want: []string{"AAPL", "MSFT"}},
		{name: "mixed case", env: "Aapl,msFT", want: []string{"AAPL", "MSFT"}},
		{name: "trailing comma", env: "AAPL,", want: []string{"AAPL"}},
		{name: "only commas", env: ",,", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_CSV", tt.env)
			got := csvToSlice("TEST_CSV")
			if tt.want == nil {
				if got != nil {
					t.Errorf("csvToSlice(%q) = %v, want nil", tt.env, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("csvToSlice(%q) = %v, want %v", tt.env, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("csvToSlice(%q)[%d] = %q, want %q", tt.env, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "set", value: "something", wantErr: false},
		{name: "empty", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_REQUIRED", tt.value)
			got, err := required("TEST_REQUIRED")
			if tt.wantErr {
				if err == nil {
					t.Fatal("required() error = nil, want error")
				}
				if !strings.Contains(err.Error(), "TEST_REQUIRED is required") {
					t.Errorf("error = %q, want it to name the missing key", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("required() error = %v", err)
			}
			if got != tt.value {
				t.Errorf("required() = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestEnvOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback string
		want     string
	}{
		{name: "set overrides fallback", value: "custom", fallback: "fb", want: "custom"},
		{name: "empty uses fallback", value: "", fallback: "fb", want: "fb"},
		{name: "whitespace is kept", value: " ", fallback: "fb", want: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_ENV_DEFAULT", tt.value)
			if got := envOrDefault("TEST_ENV_DEFAULT", tt.fallback); got != tt.want {
				t.Errorf("envOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIntOrDefault(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "valid int", value: "12", fallback: 4, want: 12},
		{name: "empty uses fallback", value: "", fallback: 4, want: 4},
		{name: "non-numeric uses fallback", value: "abc", fallback: 4, want: 4},
		{name: "negative", value: "-3", fallback: 4, want: -3},
		{name: "zero", value: "0", fallback: 4, want: 0},
		{name: "float is invalid", value: "1.5", fallback: 4, want: 4},
		{name: "whitespace is invalid", value: " 7 ", fallback: 4, want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_INT_DEFAULT", tt.value)
			if got := intOrDefault("TEST_INT_DEFAULT", tt.fallback); got != tt.want {
				t.Errorf("intOrDefault() = %d, want %d", got, tt.want)
			}
		})
	}
}

// clearEnv unsets every variable the config loaders read, so each test starts clean.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DATABASE_URL", "API_PORT", "APP_ENV", "DB_POOL_MAX_CONNS",
		"MASSIVE_API_KEY", "SQS_QUEUE_URL", "TICKER_LIMIT", "TICKER_ALLOWLIST", "SFN_ARN",
	} {
		t.Setenv(k, "")
	}
}

func TestLoad(t *testing.T) {
	t.Run("defaults applied", func(t *testing.T) {
		clearEnv(t)
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
		clearEnv(t)
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
		clearEnv(t)
		if _, err := Load(); err == nil {
			t.Fatal("Load() error = nil, want error for missing DATABASE_URL")
		}
	})
}

func TestLoadFetchTickers(t *testing.T) {
	t.Run("success with defaults", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MASSIVE_API_KEY", "key")
		t.Setenv("SQS_QUEUE_URL", "https://sqs/q")

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
	})

	t.Run("allowlist and limit parsed", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MASSIVE_API_KEY", "key")
		t.Setenv("SQS_QUEUE_URL", "https://sqs/q")
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
			{name: "no api key", env: map[string]string{"SQS_QUEUE_URL": "https://sqs/q"}},
			{name: "no queue url", env: map[string]string{"MASSIVE_API_KEY": "key"}},
			{name: "neither", env: map[string]string{}},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				clearEnv(t)
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

func TestLoadStartPipeline(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		clearEnv(t)
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
				clearEnv(t)
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

// dbAndKeyLoader describes the five loaders that all read DATABASE_URL,
// MASSIVE_API_KEY and DB_POOL_MAX_CONNS into an identically shaped config.
type dbAndKeyLoader struct {
	name string
	load func() (dbURL, apiKey string, poolMax int, err error)
}

func dbAndKeyLoaders() []dbAndKeyLoader {
	return []dbAndKeyLoader{
		{name: "LoadIngestOHLCV", load: func() (string, string, int, error) {
			c, err := LoadIngestOHLCV()
			if err != nil {
				return "", "", 0, err
			}
			return c.DatabaseURL, c.MassiveAPIKey, c.PoolMaxConns, nil
		}},
		{name: "LoadFetchTechnicals", load: func() (string, string, int, error) {
			c, err := LoadFetchTechnicals()
			if err != nil {
				return "", "", 0, err
			}
			return c.DatabaseURL, c.MassiveAPIKey, c.PoolMaxConns, nil
		}},
		{name: "LoadFetchFundamentals", load: func() (string, string, int, error) {
			c, err := LoadFetchFundamentals()
			if err != nil {
				return "", "", 0, err
			}
			return c.DatabaseURL, c.MassiveAPIKey, c.PoolMaxConns, nil
		}},
		{name: "LoadEnrichTicker", load: func() (string, string, int, error) {
			c, err := LoadEnrichTicker()
			if err != nil {
				return "", "", 0, err
			}
			return c.DatabaseURL, c.MassiveAPIKey, c.PoolMaxConns, nil
		}},
		{name: "LoadComputeStats", load: func() (string, string, int, error) {
			c, err := LoadComputeStats()
			if err != nil {
				return "", "", 0, err
			}
			return c.DatabaseURL, c.MassiveAPIKey, c.PoolMaxConns, nil
		}},
	}
}

func TestDBAndKeyLoaders_Success(t *testing.T) {
	for _, l := range dbAndKeyLoaders() {
		t.Run(l.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost/db")
			t.Setenv("MASSIVE_API_KEY", "key")

			dbURL, apiKey, poolMax, err := l.load()
			if err != nil {
				t.Fatalf("%s() error = %v", l.name, err)
			}
			if dbURL != "postgres://localhost/db" {
				t.Errorf("DatabaseURL = %q", dbURL)
			}
			if apiKey != "key" {
				t.Errorf("MassiveAPIKey = %q", apiKey)
			}
			if poolMax != 1 {
				t.Errorf("PoolMaxConns = %d, want default 1", poolMax)
			}
		})
	}
}

func TestDBAndKeyLoaders_PoolOverride(t *testing.T) {
	for _, l := range dbAndKeyLoaders() {
		t.Run(l.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("DATABASE_URL", "postgres://localhost/db")
			t.Setenv("MASSIVE_API_KEY", "key")
			t.Setenv("DB_POOL_MAX_CONNS", "9")

			_, _, poolMax, err := l.load()
			if err != nil {
				t.Fatalf("%s() error = %v", l.name, err)
			}
			if poolMax != 9 {
				t.Errorf("PoolMaxConns = %d, want 9", poolMax)
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
				clearEnv(t)
				for k, v := range c.env {
					t.Setenv(k, v)
				}
				if _, _, _, err := l.load(); err == nil {
					t.Errorf("%s() error = nil, want error", l.name)
				}
			})
		}
	}
}

func TestLoadClosePipeline(t *testing.T) {
	t.Run("success with default pool", func(t *testing.T) {
		clearEnv(t)
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
		clearEnv(t)
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
		clearEnv(t)
		if _, err := LoadClosePipeline(); err == nil {
			t.Fatal("LoadClosePipeline() error = nil, want error")
		}
	})
}
