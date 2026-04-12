package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/db"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/massive"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
)

// ohlcvFetcher abstracts the Massive client for testing.
type ohlcvFetcher interface {
	FetchDailyOHLCV(ctx context.Context, ticker string, date time.Time) (*domain.DailyPrice, error)
}

// priceWriter abstracts the repository for testing.
type priceWriter interface {
	Upsert(ctx context.Context, price *domain.DailyPrice) error
}

// Response is the output payload for the IngestOHLCV Lambda.
type Response struct {
	Ticker string `json:"ticker"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

// ingestOHLCV is the core logic: fetch OHLCV then upsert to DB.
func ingestOHLCV(ctx context.Context, event pipeline.TickerEvent, fetcher ohlcvFetcher, writer priceWriter, logger *slog.Logger) (*Response, error) {
	if event.TickerID == "" {
		return nil, fmt.Errorf("ticker_id is required")
	}
	if event.Ticker == "" {
		return nil, fmt.Errorf("ticker is required")
	}

	date, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", event.Date, err)
	}

	logger.Info("fetching OHLCV", "ticker", event.Ticker, "date", event.Date)
	price, err := fetcher.FetchDailyOHLCV(ctx, event.Ticker, date)
	if err != nil {
		return nil, fmt.Errorf("fetching OHLCV for %s: %w", event.Ticker, err)
	}

	price.TickerID = event.TickerID

	logger.Info("upserting daily price", "ticker", event.Ticker, "date", event.Date)
	if err := writer.Upsert(ctx, price); err != nil {
		return nil, fmt.Errorf("upserting price for %s: %w", event.Ticker, err)
	}

	logger.Info("ingest-ohlcv complete", "ticker", event.Ticker, "date", event.Date)
	return &Response{
		Ticker: event.Ticker,
		Date:   event.Date,
		Status: "ok",
	}, nil
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadIngestOHLCV()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	client := massive.NewClient(cfg.MassiveAPIKey, logger)
	repo := repository.NewDailyPriceRepo(pool, logger)

	return ingestOHLCV(ctx, event, client, repo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
