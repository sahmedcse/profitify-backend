package testutil

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClearEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://should-be-cleared/db")
	t.Setenv("MASSIVE_API_KEY", "should-be-cleared")

	ClearEnv(t)

	for _, k := range lambdaEnvKeys {
		if v := os.Getenv(k); v != "" {
			t.Errorf("%s = %q after ClearEnv, want empty", k, v)
		}
	}
}

func TestClosedPortAddr(t *testing.T) {
	addr := ClosedPortAddr(t)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("ClosedPortAddr() = %q, not a host:port pair: %v", addr, err)
	}
	if host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", host)
	}
	if port == "0" {
		t.Error("port should be a concrete port, not 0")
	}

	// The whole point is that connecting is refused quickly rather than hanging.
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected the connection to be refused")
	}
}

func TestClosedPortAddr_ReturnsDistinctPorts(t *testing.T) {
	first := ClosedPortAddr(t)
	second := ClosedPortAddr(t)
	if first == second {
		t.Errorf("consecutive calls handed back the same port twice: %s", first)
	}
}

func TestUnreachableDSN(t *testing.T) {
	dsn := UnreachableDSN(t)

	for _, want := range []string{
		"postgres://user:pass@127.0.0.1:",
		"/testdb",
		"sslmode=disable",
		"connect_timeout=2",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("UnreachableDSN() = %q, want it to contain %q", dsn, want)
		}
	}
}
