package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/profitify/profitify-backend/internal/testutil"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// run blocks serving traffic once startup succeeds, so these tests pin the two
// startup failures. The serving and graceful-shutdown paths need a live
// database and are exercised by running the binary.

func TestRun_MissingConfig(t *testing.T) {
	testutil.ClearEnv(t)

	err := run(discardLogger())
	if err == nil {
		t.Fatal("run() error = nil, want a config error")
	}
	if !strings.Contains(err.Error(), "loading config") {
		t.Errorf("error = %q, want it wrapped with \"loading config\"", err.Error())
	}
}

func TestRun_DatabaseUnreachable(t *testing.T) {
	testutil.ClearEnv(t)
	t.Setenv("DATABASE_URL", testutil.UnreachableDSN(t))

	err := run(discardLogger())
	if err == nil {
		t.Fatal("run() error = nil, want a connection error")
	}
	if !strings.Contains(err.Error(), "connecting to database") {
		t.Errorf("error = %q, want it wrapped with \"connecting to database\"", err.Error())
	}
}
