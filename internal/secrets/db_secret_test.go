package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDBSecret_Credentials_ReturnsUsernameAndPassword(t *testing.T) {
	const password = `p#a%ss:w/rd?x@y&z=1`
	raw := fmt.Sprintf(`{"username":"test_user","password":%q}`, password)
	api := okStub(raw)

	s := NewDBSecret("arn:aws:secretsmanager:us-east-1:1:secret:db", api)
	username, gotPassword, err := s.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials() error = %v", err)
	}
	if username != "test_user" {
		t.Errorf("username = %q, want test_user", username)
	}
	if gotPassword != password {
		t.Errorf("password = %q, want %q", gotPassword, password)
	}
}

func TestDBSecret_Credentials_PassesSecretID(t *testing.T) {
	const arn = "arn:aws:secretsmanager:us-east-1:1:secret:db-admin"
	api := okStub(`{"username":"u","password":"p"}`)

	s := NewDBSecret(arn, api)
	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials() error = %v", err)
	}
}

func TestDBSecret_Credentials_CachesAfterFirstFetch(t *testing.T) {
	api := okStub(`{"username":"u","password":"p"}`)
	s := NewDBSecret("arn", api)

	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("first Credentials() error = %v", err)
	}
	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("second Credentials() error = %v", err)
	}

	if got := api.callCount(); got != 1 {
		t.Errorf("GetSecretValue called %d times, want 1", got)
	}
}

func TestDBSecret_Invalidate_ForcesRefetch(t *testing.T) {
	api := okStub(`{"username":"u","password":"p"}`)
	s := NewDBSecret("arn", api)

	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("first Credentials() error = %v", err)
	}
	s.Invalidate()
	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("second Credentials() error = %v", err)
	}

	if got := api.callCount(); got != 2 {
		t.Errorf("GetSecretValue called %d times, want 2 after Invalidate", got)
	}
}

func TestDBSecret_Credentials_DoesNotCacheFailures(t *testing.T) {
	api := &stubAPI{results: []stubResult{
		{err: errors.New("throttled")},
		{secretString: strPtr(`{"username":"u","password":"p"}`)},
	}}
	s := NewDBSecret("arn", api)

	if _, _, err := s.Credentials(context.Background()); err == nil {
		t.Fatal("first Credentials() error = nil, want error")
	}
	username, password, err := s.Credentials(context.Background())
	if err != nil {
		t.Fatalf("second Credentials() error = %v", err)
	}
	if username != "u" || password != "p" {
		t.Errorf("Credentials() = %q, %q, want u, p", username, password)
	}
	if got := api.callCount(); got != 2 {
		t.Errorf("GetSecretValue called %d times, want 2 (failure must not be cached)", got)
	}
}

func TestDBSecret_Credentials_AppliesFetchTimeout(t *testing.T) {
	api := okStub(`{"username":"u","password":"p"}`)
	s := NewDBSecret("arn", api)

	if _, _, err := s.Credentials(context.Background()); err != nil {
		t.Fatalf("Credentials() error = %v", err)
	}

	deadline, ok := api.lastCtx.Deadline()
	if !ok {
		t.Fatal("GetSecretValue was called without a context deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 10*time.Second {
		t.Errorf("deadline %v away, want (0, 10s]", remaining)
	}
}

func TestDBSecret_Credentials_ErrorsNeverContainSecret(t *testing.T) {
	const sentinel = "sw0rdf1sh0penSesame"
	half1, half2 := sentinel[:9], sentinel[9:]

	tests := []struct {
		name  string
		stub  *stubAPI
		check func(t *testing.T, err error)
	}{
		{
			name: "api error",
			stub: &stubAPI{results: []stubResult{{err: errors.New("network unreachable")}}},
		},
		{
			name: "nil SecretString",
			stub: &stubAPI{results: []stubResult{{secretString: nil}}},
		},
		{
			name: "invalid JSON containing sentinel",
			stub: okStub(fmt.Sprintf(`{not valid json %s`, sentinel)),
		},
		{
			name: "non-string password",
			stub: okStub(fmt.Sprintf(`{"username":"u","password":{"nested":%q}}`, sentinel)),
		},
		{
			name: "missing username",
			stub: okStub(fmt.Sprintf(`{"username":"","password":%q}`, sentinel)),
		},
		{
			name: "missing password",
			stub: okStub(fmt.Sprintf(`{"username":%q,"password":""}`, sentinel)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDBSecret("arn", tt.stub)
			_, _, err := s.Credentials(context.Background())
			if err == nil {
				t.Fatal("Credentials() error = nil, want error")
			}
			msg := err.Error()
			if !strings.HasPrefix(msg, "secrets:") {
				t.Errorf("error = %q, want it to start with 'secrets:'", msg)
			}
			if strings.Contains(msg, sentinel) || strings.Contains(msg, half1) || strings.Contains(msg, half2) {
				t.Errorf("error = %q, leaked the secret value", msg)
			}
		})
	}
}
