package secrets

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestStringSecret_Value_ReturnsTrimmedValue(t *testing.T) {
	api := okStub("  mk_live_ABC\n")
	s := NewStringSecret("arn", api)

	got, err := s.Value(context.Background())
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if got != "mk_live_ABC" {
		t.Errorf("Value() = %q, want mk_live_ABC", got)
	}
}

func TestStringSecret_Value_CachesAfterFirstFetch(t *testing.T) {
	api := okStub("mk_live_ABC")
	s := NewStringSecret("arn", api)

	if _, err := s.Value(context.Background()); err != nil {
		t.Fatalf("first Value() error = %v", err)
	}
	if _, err := s.Value(context.Background()); err != nil {
		t.Fatalf("second Value() error = %v", err)
	}

	if got := api.callCount(); got != 1 {
		t.Errorf("GetSecretValue called %d times, want 1", got)
	}
}

func TestStringSecret_Invalidate_ForcesRefetch(t *testing.T) {
	api := okStub("mk_live_ABC")
	s := NewStringSecret("arn", api)

	if _, err := s.Value(context.Background()); err != nil {
		t.Fatalf("first Value() error = %v", err)
	}
	s.Invalidate()
	if _, err := s.Value(context.Background()); err != nil {
		t.Fatalf("second Value() error = %v", err)
	}

	if got := api.callCount(); got != 2 {
		t.Errorf("GetSecretValue called %d times, want 2 after Invalidate", got)
	}
}

func TestStringSecret_Value_DoesNotCacheFailures(t *testing.T) {
	api := &stubAPI{results: []stubResult{
		{err: errors.New("throttled")},
		{secretString: strPtr("mk_live_ABC")},
	}}
	s := NewStringSecret("arn", api)

	if _, err := s.Value(context.Background()); err == nil {
		t.Fatal("first Value() error = nil, want error")
	}
	got, err := s.Value(context.Background())
	if err != nil {
		t.Fatalf("second Value() error = %v", err)
	}
	if got != "mk_live_ABC" {
		t.Errorf("Value() = %q, want mk_live_ABC", got)
	}
	if callCount := api.callCount(); callCount != 2 {
		t.Errorf("GetSecretValue called %d times, want 2 (failure must not be cached)", callCount)
	}
}

func TestStringSecret_Value_AppliesFetchTimeout(t *testing.T) {
	api := okStub("mk_live_ABC")
	s := NewStringSecret("arn", api)

	if _, err := s.Value(context.Background()); err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	deadline, ok := api.lastCtx.Deadline()
	if !ok {
		t.Fatal("GetSecretValue was called without a context deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 10*time.Second {
		t.Errorf("deadline %v away, want (0, 10s]", remaining)
	}
}

func TestStringSecret_Value_ErrorsNeverContainSecret(t *testing.T) {
	const sentinel = "mk_live_shouldNotLeak"
	half1, half2 := sentinel[:10], sentinel[10:]

	tests := []struct {
		name string
		stub *stubAPI
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
			name: "whitespace only",
			stub: okStub("   \n\t  "),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStringSecret("arn", tt.stub)
			_, err := s.Value(context.Background())
			if err == nil {
				t.Fatal("Value() error = nil, want error")
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
