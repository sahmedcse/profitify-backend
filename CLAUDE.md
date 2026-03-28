# Profitify Backend

Go backend for the Profitify financial analytics platform. Provides AWS Lambda functions, cron services, an HTTP API, and TimescaleDB migrations.

## Tech Stack

- **Go 1.23+** — primary language
- **Chi** — HTTP router (`github.com/go-chi/chi/v5`)
- **pgx/v5** — PostgreSQL driver with connection pooling (`pgxpool`)
- **goose** — database migrations (`github.com/pressly/goose/v3`)
- **aws-lambda-go** — AWS Lambda runtime
- **TimescaleDB** — time-series PostgreSQL extension
- **slog** — structured JSON logging (stdlib)

## Directory Structure

```
cmd/                    Entrypoints (one main.go per binary)
  api/                  HTTP API server
  cron/                 Cron job runner
  lambda-*/             Lambda functions (one per handler)
internal/               Private application code
  api/                  Router and HTTP handler wiring
    handler/            HTTP handlers
  config/               Environment variable loader
  db/                   Database pool + migration runner
  middleware/           HTTP middleware (logging, etc.)
  lambda/               Shared Lambda utilities
db/migrations/          SQL migration files (goose, timestamp format)
scripts/                Shell scripts (migrate.sh)
```

## Build & Run

```bash
make build              # Build all binaries → bin/
make build-api          # Build API server only
make build-lambdas      # Build Lambda (linux/arm64) → bin/lambda-*/bootstrap
make clean              # Remove bin/ and coverage files
```

## Test

```bash
make test               # Run all tests
make test-race          # Run with race detector
make test-cover         # Generate coverage report → coverage.html
```

## Lint

```bash
make lint               # Requires golangci-lint installed
make vet                # Run go vet
```

## Database Migrations

```bash
# Requires DATABASE_URL environment variable
make migrate-up         # Apply pending migrations
make migrate-down       # Roll back last migration
make migrate-status     # Show migration status
make migrate-create NAME=add_users  # Create new migration file
```

Migration files use **goose timestamp format**: `YYYYMMDDHHMMSS_<description>.sql`

## Configuration

All configuration via environment variables (12-factor):

| Variable       | Required | Default       | Description                |
|---------------|----------|---------------|----------------------------|
| `DATABASE_URL` | Yes      | —             | PostgreSQL connection string |
| `API_PORT`     | No       | `8080`        | HTTP server port           |
| `APP_ENV`      | No       | `development` | Environment name           |

## Go Conventions

- **Errors**: Always wrap with context: `fmt.Errorf("doing thing: %w", err)`
- **No ORM**: Raw SQL with pgx. Use named parameters.
- **Packages**: Keep `internal/` private. One responsibility per package.
- **Logging**: Use `log/slog` with JSON handler. Pass logger as dependency.
- **Testing**: Table-driven tests. Use `testify` for assertions if needed.
- **TDD**: Write tests first, then implementation.
- **DRY**: Extract shared code into `internal/` packages. No duplicated logic.
- **Coverage**: Maintain >90% test coverage. CI enforces this threshold.

## TimescaleDB Conventions

- Time columns: `TIMESTAMPTZ NOT NULL`
- Always set chunk interval explicitly: `SELECT create_hypertable('table', 'time', chunk_time_interval => INTERVAL '1 day');`
- Use continuous aggregates for materialized rollups
- Compression policies for data older than retention window

## Lambda Conventions

- One function per `cmd/lambda-<domain>-<action>/` directory
- Build target: `GOOS=linux GOARCH=arm64` (Graviton2)
- Binary name must be `bootstrap` for provided.al2023 runtime
- Use `internal/lambda.InitLogger()` for consistent logging
- Build tag: `-tags lambda.norpc` for response streaming support

## Branch & Commit Conventions

- **Branches**: `feature/`, `bug/`, `hotfix/`, `chore/`
- **Commits**: Conventional Commits format
  - `feat: add user authentication`
  - `fix: correct price calculation`
  - `chore: update dependencies`
  - `refactor: extract db helpers`
