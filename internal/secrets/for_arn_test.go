package secrets

import (
	"context"
	"errors"
	"testing"
)

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
