package testutil

import (
	"context"
	"sync"
)

// FakeCredentialSource is a test double for internal/db's CredentialSource
// interface. It is defined structurally — internal/testutil does not import
// internal/db — because internal/db's own tests import internal/testutil;
// importing internal/db back from here would form a cycle. Any type with
// Credentials(context.Context) (string, string, error) and Invalidate()
// satisfies db.CredentialSource, so this fake is usable wherever that
// interface is expected.
type FakeCredentialSource struct {
	// Username and Password are returned by Credentials when Err is nil.
	Username, Password string
	// Err, when set, is returned by Credentials instead of Username/Password.
	Err error

	mu              sync.Mutex
	calls           int
	invalidateCalls int
}

// Credentials returns the configured Username/Password, or Err if set. It
// counts every call so tests can assert on how many times it was invoked.
func (f *FakeCredentialSource) Credentials(context.Context) (string, string, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()

	if f.Err != nil {
		return "", "", f.Err
	}
	return f.Username, f.Password, nil
}

// Invalidate records that it was called, so tests can assert a failed
// connection attempt triggered a credential refresh.
func (f *FakeCredentialSource) Invalidate() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.invalidateCalls++
}

// Calls returns how many times Credentials has been called.
func (f *FakeCredentialSource) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// Invalidated returns how many times Invalidate has been called.
func (f *FakeCredentialSource) Invalidated() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.invalidateCalls
}
