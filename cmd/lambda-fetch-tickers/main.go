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
	"github.com/profitify/profitify-backend/internal/repository"
)

// connectDB and resolveMassiveAPIKey are variables so tests can substitute a
// fake instead of dialing a real database or calling Secrets Manager.
var (
	connectDB            = lambdautil.ConnectDB
	resolveMassiveAPIKey = lambdautil.MassiveAPIKey
)

// tickerFetcher abstracts the Massive client for testing.
type tickerFetcher interface {
	FetchActiveTickers(ctx context.Context) ([]domain.Ticker, error)
}

// tickerUpserter abstracts the ticker repository for testing.
// repository.TickerRepository satisfies this implicitly.
type tickerUpserter interface {
	UpsertBatch(ctx context.Context, tickers []domain.Ticker) error
}

// publisher abstracts the SQS publisher for testing.
type publisher interface {
	SendBatch(ctx context.Context, messages []queue.TickerMessage) error
}

// Event is the optional input payload for the FetchTickers Lambda.
type Event struct {
	Date string `json:"date"`
	// Tickers, when non-empty after normalization, replaces the env
	// TICKER_ALLOWLIST for this invoke and disables TICKER_LIMIT.
	Tickers []string `json:"tickers,omitempty"`
}

// Response is the output payload for the FetchTickers Lambda.
type Response struct {
	TickerCount int    `json:"ticker_count"`
	Date        string `json:"date"`
	// UnmatchedTickers lists requested event symbols Massive did not return
	// as active. Only populated when the event carried a ticker list.
	UnmatchedTickers []string `json:"unmatched_tickers,omitempty"`
}

// filterByAllowlist returns only tickers whose symbol is in the allowlist.
// If allowlist is empty, all tickers pass through unchanged.
func filterByAllowlist(tickers []domain.Ticker, allowlist []string) []domain.Ticker {
	if len(allowlist) == 0 {
		return tickers
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, s := range allowlist {
		allowed[s] = struct{}{}
	}
	filtered := make([]domain.Ticker, 0, len(allowlist))
	for _, t := range tickers {
		if _, ok := allowed[t.Ticker]; ok {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// effectiveTickerLimit returns 0 (unbounded) when the event carries its own
// ticker list, since TICKER_LIMIT could otherwise truncate the Massive page
// walk before a requested symbol is reached. Otherwise it returns cfgLimit.
func effectiveTickerLimit(cfgLimit int, eventTickers []string) int {
	if len(eventTickers) > 0 {
		return 0
	}
	return cfgLimit
}

// fetchAndPublish fetches active tickers from Massive, upserts the filtered
// set (assigning each a tickers.id), and publishes one SQS message per
// ticker.
func fetchAndPublish(
	ctx context.Context,
	event Event,
	fetcher tickerFetcher,
	upserter tickerUpserter,
	pub publisher,
	allowlist []string,
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

	eventTickers := config.NormalizeSymbols(event.Tickers)
	filter := allowlist
	if len(eventTickers) > 0 {
		logger.Info("event tickers override allowlist", "tickers", eventTickers)
		filter = eventTickers
	}

	if len(filter) > 0 {
		logger.Info("applying ticker filter", "filter", filter, "before", len(tickers))
		tickers = filterByAllowlist(tickers, filter)
		logger.Info("filtered tickers", "after", len(tickers))
	}

	var unmatched []string
	if len(eventTickers) > 0 {
		found := make(map[string]struct{}, len(tickers))
		for _, t := range tickers {
			found[t.Ticker] = struct{}{}
		}
		for _, sym := range eventTickers {
			if _, ok := found[sym]; !ok {
				unmatched = append(unmatched, sym)
			}
		}
		if len(unmatched) > 0 {
			logger.Info("event tickers unmatched", "unmatched_tickers", unmatched)
		}
	}

	logger.Info("upserting filtered tickers", "count", len(tickers))
	if err := upserter.UpsertBatch(ctx, tickers); err != nil {
		return nil, fmt.Errorf("upserting tickers: %w", err)
	}

	for i := range tickers {
		if tickers[i].ID == "" {
			return nil, fmt.Errorf("ticker %s has no id after upsert", tickers[i].Ticker)
		}
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
		TickerCount:      len(tickers),
		Date:             date,
		UnmatchedTickers: unmatched,
	}, nil
}

func handleRequest(ctx context.Context, event Event) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadFetchTickers()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	eventTickers := config.NormalizeSymbols(event.Tickers)
	if len(eventTickers) > 0 {
		logger.Info("event carries an explicit ticker list", "tickers", eventTickers)
	}
	limit := effectiveTickerLimit(cfg.TickerLimit, eventTickers)

	apiKey, err := resolveMassiveAPIKey(ctx, cfg.MassiveAPIKey, cfg.MassiveAPIKeySecretARN)
	if err != nil {
		return nil, fmt.Errorf("resolving Massive API key: %w", err)
	}

	client := massive.NewClient(apiKey, logger, massive.WithMaxTickers(limit))

	pub, err := queue.NewPublisher(ctx, cfg.SQSQueueURL)
	if err != nil {
		return nil, fmt.Errorf("creating SQS publisher: %w", err)
	}

	pool, err := connectDB(ctx, cfg.DatabaseURL, cfg.DBSecretARN, cfg.PoolMaxConns)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	tickerRepo := repository.NewTickerRepo(pool, logger)

	return fetchAndPublish(ctx, event, client, tickerRepo, pub, cfg.TickerAllowlist, logger)
}

func main() {
	lambda.Start(handleRequest)
}
