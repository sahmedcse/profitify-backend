package db

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

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

func TestParseConfig_InvalidConnString(t *testing.T) {
	tests := []struct {
		name    string
		connStr string
	}{
		{name: "malformed url", connStr: "://nope"},
		{name: "bad scheme", connStr: "http://localhost:5432/db"},
		{name: "unparseable port", connStr: "postgres://user:pass@localhost:notaport/db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(tt.connStr)
			if err == nil {
				t.Fatalf("ParseConfig(%q) error = nil, want error", tt.connStr)
			}
			if cfg != nil {
				t.Error("ParseConfig() returned a config alongside an error")
			}
			if !strings.Contains(err.Error(), "db: failed to parse config") {
				t.Errorf("error = %q, want it wrapped with context", err.Error())
			}
		})
	}
}

func TestParseConfig_AppliesOptionsInOrder(t *testing.T) {
	cfg, err := ParseConfig(
		"postgres://user:pass@localhost:5432/testdb",
		WithMaxConns(2),
		WithMaxConns(7),
	)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.MaxConns != 7 {
		t.Errorf("MaxConns = %d, want 7 (last option wins)", cfg.MaxConns)
	}
}

func TestWithMaxConns_NegativeIgnored(t *testing.T) {
	cfg, err := ParseConfig("postgres://user:pass@localhost:5432/testdb", WithMaxConns(-1))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if cfg.MaxConns <= 0 {
		t.Errorf("MaxConns = %d, want the pgx default to survive a negative option", cfg.MaxConns)
	}
}

func TestNew_InvalidConnString(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	pool, err := New(ctx, "://nope")
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("New() error = nil, want error for an unparseable conn string")
	}
	if pool != nil {
		t.Error("New() returned a pool alongside an error")
	}
	if !strings.Contains(err.Error(), "db: failed to parse config") {
		t.Errorf("error = %q, want the parse failure surfaced", err.Error())
	}
}

func TestNew_UnreachableDatabase(t *testing.T) {
	// Bind a port then close it, so the address is almost certainly refused
	// rather than firewalled (which would hang until the context expires).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot reserve a local port: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("postgres://user:pass@%s/testdb?sslmode=disable&connect_timeout=2", addr)
	pool, err := New(ctx, connStr)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("New() error = nil, want a ping failure against a closed port")
	}
	if !strings.Contains(err.Error(), "db: failed to ping database") {
		t.Errorf("error = %q, want the ping failure surfaced", err.Error())
	}
}

func TestMigrate_UnreachableDatabase(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot reserve a local port: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("postgres://user:pass@%s/testdb?sslmode=disable&connect_timeout=2", addr)
	err = Migrate(ctx, connStr, "../../db/migrations")
	if err == nil {
		t.Fatal("Migrate() error = nil, want an error against a closed port")
	}
	if !strings.Contains(err.Error(), "migrate: failed to run migrations") {
		t.Errorf("error = %q, want the migration failure wrapped", err.Error())
	}
}
