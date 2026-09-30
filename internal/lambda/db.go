package lambda

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/secrets"
)

// credentialSourceFor builds the db.CredentialSource for a DB secret ARN. It
// is a variable so tests can substitute a fake instead of calling AWS.
var credentialSourceFor = func(ctx context.Context, arn string) (db.CredentialSource, error) {
	return secrets.ForARN(ctx, arn)
}

// ConnectDB connects to the pipeline database. When secretARN is empty it
// falls back to using databaseURL as-is (local dev / docker-compose, where
// DATABASE_URL may still carry credentials). Otherwise it resolves the
// connection's username and password from Secrets Manager and expects a
// credential-free databaseURL; the secret wins over any URL userinfo.
func ConnectDB(ctx context.Context, databaseURL, secretARN string, maxConns int) (*pgxpool.Pool, error) {
	if secretARN == "" {
		return db.New(ctx, databaseURL, db.WithMaxConns(int32(maxConns)))
	}

	src, err := credentialSourceFor(ctx, secretARN)
	if err != nil {
		return nil, fmt.Errorf("db credentials: %w", err)
	}

	return db.NewWithCredentials(ctx, databaseURL, src, db.WithMaxConns(int32(maxConns)))
}
