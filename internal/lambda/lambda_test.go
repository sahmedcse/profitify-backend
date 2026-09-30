package lambda

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/testutil"
)

func TestInitLogger_ReturnsUsableLogger(t *testing.T) {
	logger := InitLogger()
	if logger == nil {
		t.Fatal("InitLogger() returned nil")
	}

	// Must not panic when used.
	logger.Info("test message", slog.String("key", "value"))
}

func TestInitLogger_LevelThresholds(t *testing.T) {
	handler := InitLogger().Handler()

	tests := []struct {
		name    string
		level   slog.Level
		enabled bool
	}{
		{name: "debug suppressed", level: slog.LevelDebug, enabled: false},
		{name: "info enabled", level: slog.LevelInfo, enabled: true},
		{name: "warn enabled", level: slog.LevelWarn, enabled: true},
		{name: "error enabled", level: slog.LevelError, enabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.Enabled(t.Context(), tt.level); got != tt.enabled {
				t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.enabled)
			}
		})
	}
}

func TestInitLogger_UsesJSONHandler(t *testing.T) {
	if _, ok := InitLogger().Handler().(*slog.JSONHandler); !ok {
		t.Errorf("handler type = %T, want *slog.JSONHandler", InitLogger().Handler())
	}
}

func TestInitLogger_ReturnsIndependentInstances(t *testing.T) {
	first := InitLogger()
	second := InitLogger()
	if first == second {
		t.Error("InitLogger() should return a new logger each call")
	}
}

// fakeCredSource is a test db.CredentialSource.
type fakeCredSource struct {
	username, password string
	err                 error
}

func (f *fakeCredSource) Credentials(context.Context) (string, string, error) {
	if f.err != nil {
		return "", "", f.err
	}
	return f.username, f.password, nil
}

func (f *fakeCredSource) Invalidate() {}

// fakeValueSource is a test valueSource.
type fakeValueSource struct {
	value string
	err   error
}

func (f *fakeValueSource) Value(context.Context) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

func TestConnectDB_NoSecretARN_UsesDatabaseURLAsIs(t *testing.T) {
	called := false
	orig := credentialSourceFor
	credentialSourceFor = func(context.Context, string) (db.CredentialSource, error) {
		called = true
		return &fakeCredSource{}, nil
	}
	t.Cleanup(func() { credentialSourceFor = orig })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	pool, err := ConnectDB(ctx, testutil.UnreachableDSN(t), "", 1)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("ConnectDB() error = nil, want a ping failure against a closed port")
	}
	if !strings.Contains(err.Error(), "db: failed to ping database") {
		t.Errorf("error = %q, want the plain db.New failure surfaced", err.Error())
	}
	if called {
		t.Error("credentialSourceFor was called despite secretARN being empty")
	}
}

func TestConnectDB_WithSecretARN_UsesCredentialSource(t *testing.T) {
	var gotARN string
	orig := credentialSourceFor
	credentialSourceFor = func(_ context.Context, arn string) (db.CredentialSource, error) {
		gotARN = arn
		return &fakeCredSource{username: "u", password: "p"}, nil
	}
	t.Cleanup(func() { credentialSourceFor = orig })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	connStr := fmt.Sprintf("postgres://%s/testdb?sslmode=disable&connect_timeout=2", testutil.ClosedPortAddr(t))
	pool, err := ConnectDB(ctx, connStr, "arn:secret:db", 1)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("ConnectDB() error = nil, want a ping failure against a closed port")
	}
	if !strings.Contains(err.Error(), "db: failed to ping database") {
		t.Errorf("error = %q, want the credentialed connect failure surfaced", err.Error())
	}
	if gotARN != "arn:secret:db" {
		t.Errorf("credentialSourceFor called with arn = %q, want arn:secret:db", gotARN)
	}
}

func TestConnectDB_WithSecretARN_FactoryError(t *testing.T) {
	orig := credentialSourceFor
	credentialSourceFor = func(context.Context, string) (db.CredentialSource, error) {
		return nil, errors.New("no AWS config found")
	}
	t.Cleanup(func() { credentialSourceFor = orig })

	pool, err := ConnectDB(t.Context(), "postgres://db-host/profitify?sslmode=disable", "arn:secret:db", 1)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("ConnectDB() error = nil, want a wrapped factory error")
	}
	if !strings.Contains(err.Error(), "db credentials:") {
		t.Errorf("error = %q, want it wrapped as 'db credentials: ...'", err.Error())
	}
}

func TestMassiveAPIKey_NoSecretARN_ReturnsEnvValue(t *testing.T) {
	called := false
	orig := apiKeySourceFor
	apiKeySourceFor = func(context.Context, string) (valueSource, error) {
		called = true
		return &fakeValueSource{value: "should-not-be-used"}, nil
	}
	t.Cleanup(func() { apiKeySourceFor = orig })

	got, err := MassiveAPIKey(context.Background(), "env-value", "")
	if err != nil {
		t.Fatalf("MassiveAPIKey() error = %v", err)
	}
	if got != "env-value" {
		t.Errorf("MassiveAPIKey() = %q, want env-value", got)
	}
	if called {
		t.Error("apiKeySourceFor was called despite secretARN being empty")
	}
}

func TestMassiveAPIKey_WithSecretARN_ReturnsSecretValue(t *testing.T) {
	var gotARN string
	orig := apiKeySourceFor
	apiKeySourceFor = func(_ context.Context, arn string) (valueSource, error) {
		gotARN = arn
		return &fakeValueSource{value: "secret-value"}, nil
	}
	t.Cleanup(func() { apiKeySourceFor = orig })

	got, err := MassiveAPIKey(context.Background(), "env-value", "arn:secret:massive")
	if err != nil {
		t.Fatalf("MassiveAPIKey() error = %v", err)
	}
	if got != "secret-value" {
		t.Errorf("MassiveAPIKey() = %q, want secret-value (the secret must win)", got)
	}
	if gotARN != "arn:secret:massive" {
		t.Errorf("apiKeySourceFor called with arn = %q, want arn:secret:massive", gotARN)
	}
}

func TestMassiveAPIKey_SourceOrFactoryError_Wrapped(t *testing.T) {
	const envValue = "env-should-not-leak"
	const secretValue = "secret-should-not-leak"

	tests := []struct {
		name string
		src  func(ctx context.Context, arn string) (valueSource, error)
	}{
		{
			name: "factory error",
			src: func(context.Context, string) (valueSource, error) {
				return nil, errors.New("no AWS config found")
			},
		},
		{
			name: "value error",
			src: func(context.Context, string) (valueSource, error) {
				return &fakeValueSource{err: errors.New("access denied")}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := apiKeySourceFor
			apiKeySourceFor = tt.src
			t.Cleanup(func() { apiKeySourceFor = orig })

			_, err := MassiveAPIKey(context.Background(), envValue, "arn:secret:massive")
			if err == nil {
				t.Fatal("MassiveAPIKey() error = nil, want error")
			}
			msg := err.Error()
			if !strings.Contains(msg, "massive api key:") {
				t.Errorf("error = %q, want it wrapped as 'massive api key: ...'", msg)
			}
			if strings.Contains(msg, envValue) {
				t.Errorf("error = %q, leaked the env value", msg)
			}
			if strings.Contains(msg, secretValue) {
				t.Errorf("error = %q, leaked the secret value", msg)
			}
		})
	}
}
