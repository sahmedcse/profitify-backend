.PHONY: build build-api build-cron build-lambdas lint test test-race test-cover migrate-up migrate-down migrate-status migrate-create clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
BINARY_DIR=bin
MIGRATIONS_DIR=db/migrations

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## build: Build all binaries
build: build-api build-cron build-lambdas

## build-api: Build the API server
build-api:
	$(GOBUILD) -o $(BINARY_DIR)/api ./cmd/api

## build-cron: Build the cron runner
build-cron:
	$(GOBUILD) -o $(BINARY_DIR)/cron ./cmd/cron

## build-lambdas: Build all Lambda functions (linux/arm64 for Graviton2)
build-lambdas:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -tags lambda.norpc -o $(BINARY_DIR)/lambda-example/bootstrap ./cmd/lambda-example

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## test: Run all tests
test:
	$(GOTEST) ./...

## test-race: Run tests with race detector
test-race:
	$(GOTEST) -race ./...

## test-cover: Run tests with coverage report
test-cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## vet: Run go vet
vet:
	$(GOVET) ./...

## migrate-up: Run all pending migrations
migrate-up:
	./scripts/migrate.sh up

## migrate-down: Roll back the last migration
migrate-down:
	./scripts/migrate.sh down

## migrate-status: Show migration status
migrate-status:
	./scripts/migrate.sh status

## migrate-create: Create a new migration (usage: make migrate-create NAME=description)
migrate-create:
	goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

## clean: Remove build artifacts
clean:
	rm -rf $(BINARY_DIR) coverage.out coverage.html

## tidy: Run go mod tidy
tidy:
	$(GOCMD) mod tidy
