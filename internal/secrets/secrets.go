// Package secrets resolves runtime credentials from AWS Secrets Manager. It
// is the only package in this module that imports service/secretsmanager;
// every other package that needs a secret value goes through the types
// defined here.
package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// fetchTimeout bounds every call to Secrets Manager.
const fetchTimeout = 10 * time.Second

// API abstracts the Secrets Manager client for testing. Mirrors the
// sqsClient pattern in internal/queue/sqs.go.
type API interface {
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

// core is the shared fetch-and-cache logic behind DBSecret and StringSecret.
// It caches the raw SecretString after the first successful fetch. Failures
// are never cached, so the next call retries against Secrets Manager.
type core struct {
	arn string
	api API

	mu     sync.Mutex
	value  string
	cached bool
}

func newCore(arn string, api API) *core {
	return &core{arn: arn, api: api}
}

// fetch returns the cached raw SecretString, fetching it under fetchTimeout
// on a cache miss.
func (c *core) fetch(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached {
		return c.value, nil
	}

	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	out, err := c.api.GetSecretValue(fetchCtx, &secretsmanager.GetSecretValueInput{SecretId: &c.arn})
	if err != nil {
		return "", fmt.Errorf("secrets: get secret value for %s: %w", c.arn, err)
	}
	if out.SecretString == nil {
		return "", fmt.Errorf("secrets: secret %s has no SecretString", c.arn)
	}

	c.value = *out.SecretString
	c.cached = true
	return c.value, nil
}

// invalidate clears the cache, forcing the next fetch call to refetch.
func (c *core) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cached = false
	c.value = ""
}

// dbSecretFields is the JSON shape of the DB admin secret. Extra keys are
// ignored by encoding/json.
type dbSecretFields struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// DBSecret resolves database credentials from a JSON {username,password}
// secret in AWS Secrets Manager.
type DBSecret struct {
	core *core
}

// NewDBSecret creates a DBSecret backed by api for the given secret ARN.
func NewDBSecret(arn string, api API) *DBSecret {
	return &DBSecret{core: newCore(arn, api)}
}

// Credentials returns the username and password decoded from the secret,
// caching them after the first successful fetch. Errors never contain the
// secret value or the underlying encoding/json error text.
func (s *DBSecret) Credentials(ctx context.Context) (username, password string, err error) {
	raw, err := s.core.fetch(ctx)
	if err != nil {
		return "", "", err
	}

	var fields dbSecretFields
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return "", "", fmt.Errorf("secrets: secret %s is not a JSON object with string username and password", s.core.arn)
	}
	if fields.Username == "" {
		return "", "", fmt.Errorf("secrets: secret %s is missing %q", s.core.arn, "username")
	}
	if fields.Password == "" {
		return "", "", fmt.Errorf("secrets: secret %s is missing %q", s.core.arn, "password")
	}
	return fields.Username, fields.Password, nil
}

// Invalidate clears the cached credentials, forcing the next call to
// Credentials to refetch from Secrets Manager.
func (s *DBSecret) Invalidate() {
	s.core.invalidate()
}

// StringSecret resolves a plain-string secret (such as the Massive API key)
// from AWS Secrets Manager.
type StringSecret struct {
	core *core
}

// NewStringSecret creates a StringSecret backed by api for the given secret ARN.
func NewStringSecret(arn string, api API) *StringSecret {
	return &StringSecret{core: newCore(arn, api)}
}

// Value returns the trimmed secret value, caching it after the first
// successful fetch. A surrounding newline from an operator's paste is
// harmless. Errors never contain the secret value.
func (s *StringSecret) Value(ctx context.Context) (string, error) {
	raw, err := s.core.fetch(ctx)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("secrets: secret %s is empty", s.core.arn)
	}
	return trimmed, nil
}

// Invalidate clears the cached value, forcing the next call to Value to
// refetch from Secrets Manager.
func (s *StringSecret) Invalidate() {
	s.core.invalidate()
}

// defaultNewAPI builds the real Secrets Manager client from the default AWS
// config.
func defaultNewAPI(ctx context.Context) (API, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("secrets: loading AWS config: %w", err)
	}
	return secretsmanager.NewFromConfig(cfg), nil
}

// newAPI builds the shared Secrets Manager client. It is a variable so tests
// can substitute a fake.
var newAPI = defaultNewAPI

var (
	clientMu sync.Mutex
	client   API
)

// sharedAPI returns the process-wide Secrets Manager client, building it
// once via newAPI and sharing it between ForARN and StringSecretForARN. A
// factory error is never memoised, so the next call tries again.
func sharedAPI(ctx context.Context) (API, error) {
	clientMu.Lock()
	defer clientMu.Unlock()

	if client != nil {
		return client, nil
	}

	api, err := newAPI(ctx)
	if err != nil {
		return nil, err
	}

	client = api
	return client, nil
}

var (
	dbSecretsMu sync.Mutex
	dbSecrets   = map[string]*DBSecret{}
)

// ForARN returns the process-wide DBSecret for arn, building the shared
// client and creating the instance on first use. Subsequent calls with the
// same arn return the same instance, so there is one fetch per container
// per secret until Invalidate is called.
func ForARN(ctx context.Context, arn string) (*DBSecret, error) {
	dbSecretsMu.Lock()
	defer dbSecretsMu.Unlock()

	if s, ok := dbSecrets[arn]; ok {
		return s, nil
	}

	api, err := sharedAPI(ctx)
	if err != nil {
		return nil, fmt.Errorf("secrets: building client: %w", err)
	}

	s := NewDBSecret(arn, api)
	dbSecrets[arn] = s
	return s, nil
}

var (
	stringSecretsMu sync.Mutex
	stringSecrets   = map[string]*StringSecret{}
)

// StringSecretForARN returns the process-wide StringSecret for arn, building
// the shared client and creating the instance on first use.
func StringSecretForARN(ctx context.Context, arn string) (*StringSecret, error) {
	stringSecretsMu.Lock()
	defer stringSecretsMu.Unlock()

	if s, ok := stringSecrets[arn]; ok {
		return s, nil
	}

	api, err := sharedAPI(ctx)
	if err != nil {
		return nil, fmt.Errorf("secrets: building client: %w", err)
	}

	s := NewStringSecret(arn, api)
	stringSecrets[arn] = s
	return s, nil
}
