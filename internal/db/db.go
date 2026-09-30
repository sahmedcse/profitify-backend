package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Option configures the database connection pool.
type Option func(*pgxpool.Config)

// WithMaxConns sets the maximum number of connections in the pool.
// A value of 0 leaves the pgx default (4) unchanged.
func WithMaxConns(n int32) Option {
	return func(cfg *pgxpool.Config) {
		if n > 0 {
			cfg.MaxConns = n
		}
	}
}

// ParseConfig parses a connection string and applies options without
// connecting. Useful for testing pool configuration.
func ParseConfig(connStr string, opts ...Option) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("db: failed to parse config: %w", err)
	}

	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg, nil
}

// CredentialSource resolves database credentials at connect time, so the
// connection string itself can stay credential-free. Invalidate is called
// after a failed pool creation or ping so the next connect attempt refetches
// rather than retrying a stale or just-rotated credential.
type CredentialSource interface {
	Credentials(ctx context.Context) (username, password string, err error)
	Invalidate()
}

// applyCredentials resolves credentials from src and sets them on cfg. On
// error it returns without touching cfg.
func applyCredentials(ctx context.Context, cfg *pgxpool.Config, src CredentialSource) error {
	username, password, err := src.Credentials(ctx)
	if err != nil {
		return fmt.Errorf("db: failed to resolve credentials: %w", err)
	}

	cfg.ConnConfig.User = username
	cfg.ConnConfig.Password = password
	return nil
}

// connect creates a pgxpool connection pool from cfg and verifies
// connectivity with a ping.
func connect(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: failed to ping database: %w", err)
	}

	return pool, nil
}

// New creates a new pgxpool connection pool and verifies connectivity with a ping.
// It uses simple protocol mode for PgBouncer compatibility.
func New(ctx context.Context, connStr string, opts ...Option) (*pgxpool.Pool, error) {
	cfg, err := ParseConfig(connStr, opts...)
	if err != nil {
		return nil, err
	}

	return connect(ctx, cfg)
}

// NewWithCredentials parses connStr (which may be credential-free), resolves
// its username and password from src, and connects. src.Invalidate is called
// when the pool cannot be created or the ping fails, so a stale or
// just-rotated credential is not retried on the next attempt.
func NewWithCredentials(ctx context.Context, connStr string, src CredentialSource, opts ...Option) (*pgxpool.Pool, error) {
	cfg, err := ParseConfig(connStr, opts...)
	if err != nil {
		return nil, err
	}

	if err := applyCredentials(ctx, cfg, src); err != nil {
		return nil, err
	}

	pool, err := connect(ctx, cfg)
	if err != nil {
		src.Invalidate()
		return nil, err
	}

	return pool, nil
}
