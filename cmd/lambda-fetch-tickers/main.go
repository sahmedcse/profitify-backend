package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/domain"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/massive"
	"github.com/profitify/profitify-backend/internal/queue"
)

// tickerFetcher abstracts the Massive client for testing.
type tickerFetcher interface {
	FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error)
}

// publisher abstracts the SQS publisher for testing.
type publisher interface {
	SendBatch(ctx context.Context, messages []queue.TickerMessage) error
}

// Event is the optional input payload for the FetchTickers Lambda.
type Event struct {
	Date string `json:"date"`
}

// Response is the output payload for the FetchTickers Lambda.
type Response struct {
	TickerCount int    `json:"ticker_count"`
	Date        string `json:"date"`
}

// fetchAndPublish fetches active tickers from Massive and publishes each to SQS.
func fetchAndPublish(
	ctx context.Context,
	event Event,
	fetcher tickerFetcher,
	pub publisher,
	logger *slog.Logger,
) (*Response, error) {
	date := time.Now().UTC().Format("2006-01-02")
	if event.Date != "" {
		date = event.Date
	}

	logger.Info("fetching active tickers from Massive")
	tickers, err := fetcher.FetchActiveTickers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching tickers: %w", err)
	}

	messages := make([]queue.TickerMessage, len(tickers))
	for i, t := range tickers {
		messages[i] = queue.TickerMessage{
			Ticker: t,
			Date:   date,
		}
	}

	logger.Info("publishing tickers to SQS", "count", len(messages))
	if err := pub.SendBatch(ctx, messages); err != nil {
		return nil, fmt.Errorf("publishing to SQS: %w", err)
	}

	logger.Info("fetch-tickers complete", "ticker_count", len(tickers), "date", date)
	return &Response{
		TickerCount: len(tickers),
		Date:        date,
	}, nil
}

func handleRequest(ctx context.Context, event Event) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadFetchTickers()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	client := massive.NewClient(cfg.MassiveAPIKey, logger, massive.WithMaxTickers(cfg.TickerLimit))

	pub, err := queue.NewPublisher(ctx, cfg.SQSQueueURL)
	if err != nil {
		return nil, fmt.Errorf("creating SQS publisher: %w", err)
	}

	return fetchAndPublish(ctx, event, client, pub, logger)
}

func main() {
	lambda.Start(handleRequest)
}
