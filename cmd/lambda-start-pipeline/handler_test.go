package main

import (
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/profitify/profitify-backend/internal/testutil"
)

// handleRequest wires configuration and the database pool before delegating to
// the handler under test. Its success path needs a live database and is covered
// by the integration suite; these tests pin the two failure paths.

func TestHandleRequest_MissingConfig(t *testing.T) {
	testutil.ClearEnv(t)

	if _, err := handleRequest(t.Context(), events.SQSEvent{}); err == nil {
		t.Fatal("handleRequest() error = nil, want a config error")
	} else if !strings.Contains(err.Error(), "loading config") {
		t.Errorf("error = %q, want it wrapped with \"loading config\"", err.Error())
	}
}

func TestHandleRequest_DatabaseUnreachable(t *testing.T) {
	testutil.ClearEnv(t)
	t.Setenv("DATABASE_URL", testutil.UnreachableDSN(t))
	t.Setenv("SFN_ARN", "test-value")

	if _, err := handleRequest(t.Context(), events.SQSEvent{}); err == nil {
		t.Fatal("handleRequest() error = nil, want a connection error")
	} else if !strings.Contains(err.Error(), "connecting to database") {
		t.Errorf("error = %q, want it wrapped with \"connecting to database\"", err.Error())
	}
}
