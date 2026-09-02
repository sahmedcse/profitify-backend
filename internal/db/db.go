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

// New creates a new pgxpool connection pool and verifies connectivity with a ping.
// It uses simple protocol mode for PgBouncer compatibility.
func New(ctx context.Context, connStr string, opts ...Option) (*pgxpool.Pool, error) {
	cfg, err := ParseConfig(connStr, opts...)
	if err != nil {
		return nil, err
	}

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
