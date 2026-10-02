package secrets

import (
	"context"
	"sync"

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
