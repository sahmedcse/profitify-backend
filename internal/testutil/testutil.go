// Package testutil provides helpers shared across the repository's tests.
package testutil

import (
	"fmt"
	"net"
	"testing"
)

// lambdaEnvKeys lists every environment variable the config loaders read.
var lambdaEnvKeys = []string{
	"DATABASE_URL", "API_PORT", "APP_ENV", "DB_POOL_MAX_CONNS",
	"MASSIVE_API_KEY", "SQS_QUEUE_URL", "TICKER_LIMIT", "TICKER_ALLOWLIST", "SFN_ARN",
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
