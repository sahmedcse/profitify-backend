#!/usr/bin/env bash
set -euo pipefail

# Goose migration wrapper for profitify-backend.
# Usage: ./scripts/migrate.sh <command> [args]
# Commands: up, down, status, create <name>

MIGRATIONS_DIR="db/migrations"
GOOSE_DRIVER="postgres"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "ERROR: DATABASE_URL is not set" >&2
  exit 1
fi

COMMAND="${1:?Usage: $0 <up|down|status|create> [args]}"
shift

exec goose -dir "$MIGRATIONS_DIR" "$GOOSE_DRIVER" "$DATABASE_URL" "$COMMAND" "$@"
