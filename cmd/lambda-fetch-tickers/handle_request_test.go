package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHandleRequest_MissingSQSQueueURL(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing SQS_QUEUE_URL")
	}
}

func TestHandleRequest_MissingAPIKey(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing MASSIVE_API_KEY")
	}
}

func TestHandleRequest_MissingDatabaseURL(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "")

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

// TestHandleRequest_ResolvingMassiveAPIKeyError exercises the
// "resolving Massive API key: %w" branch added when MassiveAPIKey resolution
// moved behind lambdautil.MassiveAPIKey. Config load succeeds (both
// MASSIVE_API_KEY and SQS_QUEUE_URL/DATABASE_URL are set), so the error must
// come from resolveMassiveAPIKey itself.
func TestHandleRequest_ResolvingMassiveAPIKeyError(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	orig := resolveMassiveAPIKey
	resolveMassiveAPIKey = func(context.Context, string, string) (string, error) {
		return "", errors.New("secrets: get secret value for arn: access denied")
	}
	t.Cleanup(func() { resolveMassiveAPIKey = orig })

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("handleRequest() error = nil, want the Massive key resolution error")
	}
	if !strings.Contains(err.Error(), "resolving Massive API key:") {
		t.Errorf("error = %q, want it wrapped as 'resolving Massive API key: ...'", err.Error())
	}
}

// TestHandleRequest_ConnectDBError_AfterMassiveKeySucceeds exercises the
// path from a successful Massive key resolution through to ConnectDB's own
// error branch, via a faked connectDB so no real database is dialed.
func TestHandleRequest_ConnectDBError_AfterMassiveKeySucceeds(t *testing.T) {
	t.Setenv("MASSIVE_API_KEY", "test-key")
	t.Setenv("SQS_QUEUE_URL", "https://sqs.us-east-1.amazonaws.com/123/test")
	t.Setenv("DATABASE_URL", "postgres://localhost/db")

	origKey := resolveMassiveAPIKey
	resolveMassiveAPIKey = func(context.Context, string, string) (string, error) {
		return "resolved-key", nil
	}
	t.Cleanup(func() { resolveMassiveAPIKey = origKey })

	origConnect := connectDB
	connectDB = func(context.Context, string, string, int) (*pgxpool.Pool, error) {
		return nil, errors.New("db: failed to ping database: connection refused")
	}
	t.Cleanup(func() { connectDB = origConnect })

	_, err := handleRequest(context.Background(), Event{})
	if err == nil {
		t.Fatal("handleRequest() error = nil, want the ConnectDB error")
	}
	if !strings.Contains(err.Error(), "connecting to database:") {
		t.Errorf("error = %q, want it wrapped as 'connecting to database: ...'", err.Error())
	}
}
