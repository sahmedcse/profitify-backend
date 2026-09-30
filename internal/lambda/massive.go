package lambda

import (
	"context"
	"fmt"

	"github.com/profitify/profitify-backend/internal/secrets"
)

// valueSource resolves a plain-string secret value. Satisfied by
// *secrets.StringSecret.
type valueSource interface {
	Value(ctx context.Context) (string, error)
}

// apiKeySourceFor builds the valueSource for a Massive API key secret ARN.
// It is a variable so tests can substitute a fake instead of calling AWS.
var apiKeySourceFor = func(ctx context.Context, arn string) (valueSource, error) {
	return secrets.StringSecretForARN(ctx, arn)
}

// MassiveAPIKey resolves the Massive API key to use. When secretARN is
// empty it returns envValue as-is: the fallback for local dev and
// docker-compose, where the config loader already guaranteed one is set.
// Otherwise the key is resolved from Secrets Manager; when both are set,
// the secret wins.
func MassiveAPIKey(ctx context.Context, envValue, secretARN string) (string, error) {
	if secretARN == "" {
		return envValue, nil
	}

	src, err := apiKeySourceFor(ctx, secretARN)
	if err != nil {
		return "", fmt.Errorf("massive api key: %w", err)
	}

	value, err := src.Value(ctx)
	if err != nil {
		return "", fmt.Errorf("massive api key: %w", err)
	}

	return value, nil
}
