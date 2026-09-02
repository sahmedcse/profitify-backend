package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestParseConfig_SimpleProtocol(t *testing.T) {
	cfg, err := ParseConfig("postgres://user:pass@localhost:5432/testdb")
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
		t.Errorf("DefaultQueryExecMode = %v, want %v",
			cfg.ConnConfig.DefaultQueryExecMode, pgx.QueryExecModeSimpleProtocol)
	}
}

func TestParseConfig_MaxConns(t *testing.T) {
	cfg, err := ParseConfig("postgres://user:pass@localhost:5432/testdb", WithMaxConns(2))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	if cfg.MaxConns != 2 {
		t.Errorf("MaxConns = %d, want 2", cfg.MaxConns)
	}
}

func TestParseConfig_MaxConnsZero_UsesDefault(t *testing.T) {
	cfg, err := ParseConfig("postgres://user:pass@localhost:5432/testdb", WithMaxConns(0))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	// MaxConns=0 means use pgx default (4), so we don't override it
	if cfg.MaxConns == 0 {
		t.Error("MaxConns should not be 0 after ParseConfig with default pgx behavior")
	}
}
