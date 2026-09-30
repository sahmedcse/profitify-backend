package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"

	"github.com/profitify/profitify-backend/internal/db"
)

// var _ db.CredentialSource = (*DBSecret)(nil) pins DBSecret's method set to
// the shape internal/db depends on.
var _ db.CredentialSource = (*DBSecret)(nil)

// stubResult is one queued response for stubAPI.GetSecretValue.
type stubResult struct {
	secretString *string
	err          error
}

// stubAPI is a fake Secrets Manager client. Results are consumed in order;
// once exhausted, the last result repeats.
type stubAPI struct {
	mu      sync.Mutex
	calls   int
	results []stubResult
	lastCtx context.Context
}

func (s *stubAPI) GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastCtx = ctx
	idx := s.calls
	if idx >= len(s.results) {
		idx = len(s.results) - 1
	}
	s.calls++

	r := s.results[idx]
	if r.err != nil {
		return nil, r.err
	}
	return &secretsmanager.GetSecretValueOutput{SecretString: r.secretString}, nil
}

func (s *stubAPI) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func strPtr(v string) *string { return &v }

func okStub(value string) *stubAPI {
	return &stubAPI{results: []stubResult{{secretString: strPtr(value)}}}
}

// --- DBSecret ---

func TestDBSecret_Credentials_ReturnsUsernameAndPassword(t *testing.T) {
	const password = `p#a%ss:w/rd?x@y&z=1`
	raw := fmt.Sprintf(`{"username":"profitify_admin","password":%q}`, password)
	api := okStub(raw)

	s := NewDBSecret("arn:aws:secretsmanager:us-east-1:1:secret:db", api)
	username, gotPassword, err := s.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials() error = %v", err)
	}
	if username != "profitify_admin" {
		t.Errorf("username = %q, want profitify_admin", username)
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

// --- StringSecret ---

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

// --- ForARN / StringSecretForARN ---

// resetGlobalState clears the process-wide memoisation so tests don't leak
// into one another. It runs before and after each test that touches it.
func resetGlobalState(t *testing.T) {
	t.Helper()
	reset := func() {
		clientMu.Lock()
		client = nil
		clientMu.Unlock()

		dbSecretsMu.Lock()
		dbSecrets = map[string]*DBSecret{}
		dbSecretsMu.Unlock()

		stringSecretsMu.Lock()
		stringSecrets = map[string]*StringSecret{}
		stringSecretsMu.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

func TestForARN_MemoisesPerARN(t *testing.T) {
	resetGlobalState(t)

	var buildCalls int
	newAPI = func(ctx context.Context) (API, error) {
		buildCalls++
		return okStub(`{"username":"u","password":"p"}`), nil
	}
	t.Cleanup(func() {
		newAPI = defaultNewAPI
	})

	first, err := ForARN(context.Background(), "arn:db")
	if err != nil {
		t.Fatalf("first ForARN() error = %v", err)
	}
	second, err := ForARN(context.Background(), "arn:db")
	if err != nil {
		t.Fatalf("second ForARN() error = %v", err)
	}
	if first != second {
		t.Error("ForARN() with the same arn returned different instances")
	}
	if buildCalls != 1 {
		t.Errorf("newAPI called %d times, want 1", buildCalls)
	}
}

func TestStringSecretForARN_MemoisesPerARN(t *testing.T) {
	resetGlobalState(t)

	var buildCalls int
	newAPI = func(ctx context.Context) (API, error) {
		buildCalls++
		return okStub("mk_live_ABC"), nil
	}
	t.Cleanup(func() {
		newAPI = defaultNewAPI
	})

	first, err := StringSecretForARN(context.Background(), "arn:massive")
	if err != nil {
		t.Fatalf("first StringSecretForARN() error = %v", err)
	}
	second, err := StringSecretForARN(context.Background(), "arn:massive")
	if err != nil {
		t.Fatalf("second StringSecretForARN() error = %v", err)
	}
	if first != second {
		t.Error("StringSecretForARN() with the same arn returned different instances")
	}
	if buildCalls != 1 {
		t.Errorf("newAPI called %d times, want 1", buildCalls)
	}
}

func TestForARN_FactoryError(t *testing.T) {
	resetGlobalState(t)

	failing := true
	newAPI = func(ctx context.Context) (API, error) {
		if failing {
			return nil, errors.New("no AWS config found")
		}
		return okStub(`{"username":"u","password":"p"}`), nil
	}
	t.Cleanup(func() {
		newAPI = defaultNewAPI
	})

	if _, err := ForARN(context.Background(), "arn:db"); err == nil {
		t.Fatal("ForARN() error = nil, want error while the factory fails")
	}

	failing = false
	s, err := ForARN(context.Background(), "arn:db")
	if err != nil {
		t.Fatalf("ForARN() after factory recovers, error = %v", err)
	}
	if s == nil {
		t.Fatal("ForARN() returned nil instance after factory recovered")
	}
}
