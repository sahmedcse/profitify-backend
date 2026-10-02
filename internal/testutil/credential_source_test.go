package testutil

import (
	"context"
	"errors"
	"testing"
)

func TestFakeCredentialSource_Credentials_ReturnsConfiguredValues(t *testing.T) {
	f := &FakeCredentialSource{Username: "u", Password: "p"}

	username, password, err := f.Credentials(context.Background())
	if err != nil {
		t.Fatalf("Credentials() error = %v", err)
	}
	if username != "u" || password != "p" {
		t.Errorf("Credentials() = %q, %q, want u, p", username, password)
	}
	if got := f.Calls(); got != 1 {
		t.Errorf("Calls() = %d, want 1", got)
	}
}

func TestFakeCredentialSource_Credentials_ReturnsConfiguredError(t *testing.T) {
	wantErr := errors.New("secrets: get secret value for arn: boom")
	f := &FakeCredentialSource{Err: wantErr}

	_, _, err := f.Credentials(context.Background())
	if !errors.Is(err, wantErr) {
		t.Errorf("Credentials() error = %v, want %v", err, wantErr)
	}
	if got := f.Calls(); got != 1 {
		t.Errorf("Calls() = %d, want 1", got)
	}
}

func TestFakeCredentialSource_Invalidate_CountsCalls(t *testing.T) {
	f := &FakeCredentialSource{}

	if got := f.Invalidated(); got != 0 {
		t.Fatalf("Invalidated() = %d before any call, want 0", got)
	}

	f.Invalidate()
	f.Invalidate()

	if got := f.Invalidated(); got != 2 {
		t.Errorf("Invalidated() = %d, want 2", got)
	}
}
