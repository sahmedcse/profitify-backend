// Package testutil provides helpers shared across the repository's tests.
package testutil

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// lambdaEnvKeys lists every environment variable the config loaders read.
var lambdaEnvKeys = []string{
	"DATABASE_URL", "API_PORT", "APP_ENV", "DB_POOL_MAX_CONNS",
	"MASSIVE_API_KEY", "SQS_QUEUE_URL", "TICKER_LIMIT", "TICKER_ALLOWLIST", "SFN_ARN",
	"DB_SECRET_ARN", "MASSIVE_API_KEY_SECRET_ARN",
}

// ClearEnv blanks every configuration variable so a test starts from a known
// state. t.Setenv restores the previous values when the test finishes.
func ClearEnv(t *testing.T) {
	t.Helper()
	for _, k := range lambdaEnvKeys {
		t.Setenv(k, "")
	}
}

// ClosedPortAddr reserves a local TCP port and immediately releases it, so
// connections to the returned address are refused rather than left to hang.
// Using a firewalled or blackhole address instead would make callers wait for
// the context deadline on every run.
func ClosedPortAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot reserve a local port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("closing reserved port: %v", err)
	}
	return addr
}

// UnreachableDSN returns a Postgres DSN pointing at a closed local port, with a
// short connect timeout so failures surface quickly.
func UnreachableDSN(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf(
		"postgres://user:pass@%s/testdb?sslmode=disable&connect_timeout=2",
		ClosedPortAddr(t),
	)
}

// newLazyPool parses dsn and constructs a *pgxpool.Pool that has never
// dialed a real database. pgxpool.NewWithConfig only connects lazily (on
// first Acquire), so this is safe and fast to construct in tests that need
// a "successfully connected" pool without a real Postgres instance.
// Extracted from FakePool so its error paths — unreachable through a
// hardcoded, always-valid DSN — are directly testable.
func newLazyPool(dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}
	return pool, nil
}

// FakePool returns a non-nil *pgxpool.Pool that has never dialed a real
// database. It must not be used for anything that actually queries it —
// e.g. to exercise the code after a ConnectDB seam succeeds.
func FakePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := newLazyPool("postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable")
	if err != nil {
		t.Fatalf("FakePool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
