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
	"github.com/profitify/profitify-backend/internal/indicator"
	lambdautil "github.com/profitify/profitify-backend/internal/lambda"
	"github.com/profitify/profitify-backend/internal/massive"
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
)

// indicatorFetcher abstracts the Massive client for testing.
type indicatorFetcher interface {
	FetchAllIndicators(ctx context.Context, ticker string, date time.Time) (*domain.TechnicalIndicators, error)
}

// priceReader abstracts the daily price repository for testing.
type priceReader interface {
	GetByTickerAndDateRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error)
}

// technicalsWriter abstracts the technicals repository for testing.
type technicalsWriter interface {
	Upsert(ctx context.Context, tech *domain.TechnicalIndicators) error
}

// stageTracker abstracts pipeline stage tracking for testing.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// Response is the output payload for the FetchTechnicals Lambda.
type Response struct {
	Ticker            string `json:"ticker"`
	Date              string `json:"date"`
	IndicatorsFetched int    `json:"indicators_fetched"`
	SelfComputed      int    `json:"self_computed"`
}

// fetchTechnicals is the core logic.
func fetchTechnicals(ctx context.Context, event pipeline.TickerEvent, fetcher indicatorFetcher, prices priceReader, writer technicalsWriter, tracker stageTracker, logger *slog.Logger) (_ *Response, retErr error) {
	if event.TickerID == "" {
		return nil, fmt.Errorf("ticker_id is required")
	}

	date, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", event.Date, err)
	}

	st := pipeline.NewStageTracker(tracker, event.RunID, event.TickerID, domain.StageFetchTechnicals, logger)
	_ = st.Begin(ctx)
	defer func() { st.End(ctx, retErr) }()

	// 1. Fetch Massive indicators
	logger.Info("fetching indicators from Massive", "ticker", event.Ticker, "date", event.Date)
	tech, err := fetcher.FetchAllIndicators(ctx, event.Ticker, date)
	if err != nil {
		return nil, fmt.Errorf("fetching indicators for %s: %w", event.Ticker, err)
	}
	tech.TickerID = event.TickerID
	tech.Time = date

	// Count fetched indicators
	fetched := countNonNil(tech.SMA20, tech.SMA50, tech.SMA200, tech.EMA12, tech.EMA26, tech.RSI14, tech.MACDLine)

	// 2. Read OHLCV history for self-computation (last 200 days)
	from := date.AddDate(0, 0, -200)
	ohlcv, err := prices.GetByTickerAndDateRange(ctx, event.TickerID, from, date)
	if err != nil {
		logger.Warn("failed to read OHLCV history, skipping self-computed indicators", "error", err)
	}

	selfComputed := 0

	// 3. Compute Bollinger Bands (20-period, 2 std dev)
	if len(ohlcv) >= 20 {
		closes := make([]float64, len(ohlcv))
		for i, p := range ohlcv {
			closes[i] = p.Close
		}
		upper, mid, lower, err := indicator.ComputeBollinger(closes, 20, 2.0)
		if err == nil {
			tech.BollingerUpper = &upper
			tech.BollingerMiddle = &mid
			tech.BollingerLower = &lower
			selfComputed++
		} else {
			logger.Warn("bollinger computation failed", "error", err)
		}
	}

	// 4. Compute ATR (14-period)
	if len(ohlcv) >= 15 {
		atr, err := indicator.ComputeATR(ohlcv, 14)
		if err == nil {
			tech.ATR14 = &atr
			selfComputed++
		} else {
			logger.Warn("ATR computation failed", "error", err)
		}
	}

	// 5. Compute OBV
	if len(ohlcv) >= 2 {
		obv, err := indicator.ComputeOBV(ohlcv)
		if err == nil {
			tech.OBV = &obv
			selfComputed++
		} else {
			logger.Warn("OBV computation failed", "error", err)
		}
	}

	// 6. Upsert
	logger.Info("upserting technicals", "ticker", event.Ticker, "fetched", fetched, "self_computed", selfComputed)
	if err := writer.Upsert(ctx, tech); err != nil {
		return nil, fmt.Errorf("upserting technicals for %s: %w", event.Ticker, err)
	}

	return &Response{
		Ticker:            event.Ticker,
		Date:              event.Date,
		IndicatorsFetched: fetched,
		SelfComputed:      selfComputed,
	}, nil
}

func countNonNil(ptrs ...*float64) int {
	n := 0
	for _, p := range ptrs {
		if p != nil {
			n++
		}
	}
	return n
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadFetchTechnicals()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	client := massive.NewClient(cfg.MassiveAPIKey, logger)
	priceRepo := repository.NewDailyPriceRepo(pool, logger)
	techRepo := repository.NewTickerTechnicalsRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)

	return fetchTechnicals(ctx, event, client, priceRepo, techRepo, stageRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
