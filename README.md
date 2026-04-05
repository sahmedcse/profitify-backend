# Profitify Backend

Go backend for the Profitify financial analytics platform. Runs a daily market data pipeline that ingests OHLCV data for US equity tickers from [Polygon.io](https://polygon.io), enriches with fundamentals, dividends, news, and technical indicators, then computes analytics and statistics.

## Getting Started

### Prerequisites

- Go 1.23+
- Docker & Docker Compose
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2+

### Local Development

```bash
# Start TimescaleDB and run migrations
make docker-up

# Run all tests (unit + integration)
make test-integration

# Run unit tests only
make test

# Build all binaries
make build

# Open a psql shell
make docker-psql
```

### Environment Variables

| Variable       | Required | Default       | Description                  |
|----------------|----------|---------------|------------------------------|
| `DATABASE_URL` | Yes      | -             | PostgreSQL connection string |
| `API_PORT`     | No       | `8080`        | HTTP server port             |
| `APP_ENV`      | No       | `development` | Environment name             |

## Make Targets

```bash
# Build
make build                  # Build all binaries -> bin/
make build-api              # Build API server
make build-lambdas          # Build Lambdas (linux/arm64) -> bin/lambda-*/bootstrap

# Test
make test                   # Run unit tests
make test-race              # Run with race detector
make test-cover             # Generate coverage report -> coverage.html
make test-integration       # Run all tests with DATABASE_URL (requires Docker DB)

# Lint
make lint                   # Run golangci-lint
make vet                    # Run go vet

# Database
make docker-up              # Start DB + run migrations
make docker-down            # Stop DB (preserves data)
make docker-reset           # Stop DB and delete all data
make docker-psql            # Open psql shell

# Migrations
make migrate-up             # Apply pending migrations
make migrate-down           # Roll back last migration
make migrate-status         # Show migration status
make migrate-create NAME=x  # Create new migration file
```

## CI

GitHub Actions runs on every push to `main` and on pull requests:

- **Lint** - golangci-lint v2
- **Test** - unit tests with race detector + coverage
- **Integration Test** - Docker Compose with TimescaleDB
- **Build** - compile all binaries
- **Tidy Check** - verify `go.mod` / `go.sum`

All checks are required to merge.

## License

Private - All rights reserved.
