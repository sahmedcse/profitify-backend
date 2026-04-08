package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/massive"
	"github.com/profitify/profitify-backend/internal/repository"
)

// tickerFetcher abstracts the Massive client for testing.
type tickerFetcher interface {
	FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error)
}

// Response is the output payload for the FetchTickers Lambda.
type Response struct {
	TickerCount int    `json:"ticker_count"`
	Date        string `json:"date"`
}

// fetchAndUpsert is the core logic: fetch tickers then upsert to DB.
func fetchAndUpsert(ctx context.Context, fetcher tickerFetcher, repo repository.TickerRepository, logger *slog.Logger) (*Response, error) {
	logger.Info("fetching active tickers from Massive")
	tickers, err := fetcher.FetchActiveTickers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching tickers: %w", err)
	}

	logger.Info("upserting tickers to database", "count", len(tickers))
	if err := repo.UpsertBatch(ctx, tickers); err != nil {
		return nil, fmt.Errorf("upserting tickers: %w", err)
	}

	logger.Info("fetch-tickers complete", "ticker_count", len(tickers))
	return &Response{
		TickerCount: len(tickers),
		Date:        time.Now().UTC().Format("2006-01-02"),
	}, nil
}

func handleRequest(ctx context.Context) (*Response, error) {
	logger := lambdautil.InitLogger()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	apiKey := os.Getenv("MASSIVE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("MASSIVE_API_KEY is required")
	}

	pool, err := db.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	client := massive.NewClient(apiKey, logger)
	repo := repository.NewTickerRepo(pool, logger)

	return fetchAndUpsert(ctx, client, repo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
