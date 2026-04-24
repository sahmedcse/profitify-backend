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
	"github.com/profitify/profitify-backend/internal/stats"
)

// detailsFetcher abstracts the Massive client for testing.
type detailsFetcher interface {
	FetchTickerDetails(ctx context.Context, ticker string) (*domain.TickerFundamentals, error)
}

// dividendFetcher abstracts the dividend fetch for testing.
type dividendFetcher interface {
	FetchDividends(ctx context.Context, ticker string) ([]domain.TickerDividend, error)
}

// fundamentalsWriter abstracts the fundamentals repository for testing.
type fundamentalsWriter interface {
	Upsert(ctx context.Context, f *domain.TickerFundamentals) error
}

// dividendWriter abstracts the dividend repository for testing.
type dividendWriter interface {
	UpsertBatch(ctx context.Context, dividends []domain.TickerDividend) error
}

// priceReader abstracts the daily price repository for testing.
type priceReader interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.DailyPrice, error)
}

// summaryWriter abstracts the dividend summary repository for testing.
type summaryWriter interface {
	Upsert(ctx context.Context, s *domain.TickerDividendSummary) error
}

// stageTracker abstracts pipeline stage tracking for testing.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// Response is the output payload for the FetchFundamentals Lambda.
type Response struct {
	Ticker          string `json:"ticker"`
	Date            string `json:"date"`
	HasFundamentals bool   `json:"has_fundamentals"`
	DividendCount   int    `json:"dividend_count"`
}

// fetchFundamentals is the core logic.
func fetchFundamentals(
	ctx context.Context,
	event pipeline.TickerEvent,
	details detailsFetcher,
	divFetcher dividendFetcher,
	fundWriter fundamentalsWriter,
	divWriter dividendWriter,
	prices priceReader,
	sumWriter summaryWriter,
	tracker stageTracker,
	logger *slog.Logger,
) (_ *Response, retErr error) {
	if event.TickerID == "" {
		return nil, fmt.Errorf("ticker_id is required")
	}

	date, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", event.Date, err)
	}

	st := pipeline.NewStageTracker(tracker, event.RunID, event.TickerID, domain.StageFetchFundamentals, logger)
	_ = st.Begin(ctx)
	defer func() { st.End(ctx, retErr) }()

	resp := &Response{Ticker: event.Ticker, Date: event.Date}

	// 1. Fetch ticker details → upsert fundamentals
	logger.Info("fetching ticker details", "ticker", event.Ticker)
	fund, err := details.FetchTickerDetails(ctx, event.Ticker)
	if err != nil {
		logger.Warn("failed to fetch ticker details", "ticker", event.Ticker, "error", err)
	} else {
		fund.TickerID = event.TickerID
		fund.Time = date
		if err := fundWriter.Upsert(ctx, fund); err != nil {
			return nil, fmt.Errorf("upserting fundamentals for %s: %w", event.Ticker, err)
		}
		resp.HasFundamentals = true
	}

	// 2. Fetch dividends → upsert batch
	logger.Info("fetching dividends", "ticker", event.Ticker)
	dividends, err := divFetcher.FetchDividends(ctx, event.Ticker)
	if err != nil {
		logger.Warn("failed to fetch dividends", "ticker", event.Ticker, "error", err)
		return resp, nil
	}

	if len(dividends) > 0 {
		// Set TickerID on all dividends
		for i := range dividends {
			dividends[i].TickerID = event.TickerID
		}
		if err := divWriter.UpsertBatch(ctx, dividends); err != nil {
			return nil, fmt.Errorf("upserting dividends for %s: %w", event.Ticker, err)
		}
		resp.DividendCount = len(dividends)

		// 3. Compute dividend summary if we have a latest price
		latestPrice, err := prices.GetLatest(ctx, event.TickerID)
		if err != nil {
			logger.Warn("no latest price for dividend summary", "ticker", event.Ticker, "error", err)
		} else if latestPrice.Close > 0 {
			summary := stats.ComputeDividendSummary(dividends, latestPrice.Close, date)
			summary.TickerID = event.TickerID
			summary.Time = date
			if err := sumWriter.Upsert(ctx, summary); err != nil {
				logger.Warn("failed to upsert dividend summary", "ticker", event.Ticker, "error", err)
			}
		}
	}

	return resp, nil
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadFetchFundamentals()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	client := massive.NewClient(cfg.MassiveAPIKey, logger)
	fundRepo := repository.NewTickerFundamentalsRepo(pool, logger)
	divRepo := repository.NewTickerDividendRepo(pool, logger)
	priceRepo := repository.NewDailyPriceRepo(pool, logger)
	sumRepo := repository.NewTickerDividendSummaryRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)

	return fetchFundamentals(ctx, event, client, client, fundRepo, divRepo, priceRepo, sumRepo, stageRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
