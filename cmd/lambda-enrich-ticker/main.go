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
	"github.com/profitify/profitify-backend/internal/pipeline"
	"github.com/profitify/profitify-backend/internal/repository"
	"github.com/profitify/profitify-backend/internal/signal"
)

// fundamentalsReader abstracts the fundamentals repository for testing.
type fundamentalsReader interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TickerFundamentals, error)
}

// technicalsReader abstracts the technicals repository for testing.
type technicalsReader interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error)
}

// priceReader abstracts the daily price repository for testing.
type priceReader interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.DailyPrice, error)
}

// sectorUpdater abstracts the ticker repository for testing.
type sectorUpdater interface {
	UpdateSector(ctx context.Context, tickerID string, sector string) error
}

// statusUpdater abstracts the technicals repository for testing.
type statusUpdater interface {
	UpdateIndicatorStatuses(ctx context.Context, tickerID string, t time.Time, statuses map[string]string) error
}

// stageTracker abstracts pipeline stage tracking for testing.
type stageTracker interface {
	MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error)
	MarkCompleted(ctx context.Context, runID, tickerID, stage string) error
	MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error
}

// Response is the output payload for the EnrichTicker Lambda.
type Response struct {
	Ticker               string `json:"ticker"`
	Date                 string `json:"date"`
	Sector               string `json:"sector"`
	IndicatorsClassified int    `json:"indicators_classified"`
}

// enrichTicker is the core logic.
func enrichTicker(
	ctx context.Context,
	event pipeline.TickerEvent,
	fundReader fundamentalsReader,
	techReader technicalsReader,
	prices priceReader,
	sectors sectorUpdater,
	statuses statusUpdater,
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

	st := pipeline.NewStageTracker(tracker, event.RunID, event.TickerID, domain.StageEnrichTicker, logger)
	_ = st.Begin(ctx)
	defer func() { st.End(ctx, retErr) }()

	resp := &Response{Ticker: event.Ticker, Date: event.Date}

	// 1. Read latest fundamentals → map SIC code → update sector
	fund, err := fundReader.GetLatest(ctx, event.TickerID)
	if err != nil {
		logger.Warn("no fundamentals found, skipping sector mapping", "ticker", event.Ticker, "error", err)
	} else if fund.SICCode != "" {
		sector := domain.SICToSector(fund.SICCode)
		if err := sectors.UpdateSector(ctx, event.TickerID, sector); err != nil {
			return nil, fmt.Errorf("updating sector for %s: %w", event.Ticker, err)
		}
		resp.Sector = sector
		logger.Info("mapped sector", "ticker", event.Ticker, "sic", fund.SICCode, "sector", sector)
	}

	// 2. Read latest technicals + daily price → classify indicators
	tech, err := techReader.GetLatest(ctx, event.TickerID)
	if err != nil {
		logger.Warn("no technicals found, skipping classification", "ticker", event.Ticker, "error", err)
		return resp, nil
	}

	latestPrice, err := prices.GetLatest(ctx, event.TickerID)
	if err != nil {
		logger.Warn("no daily price found, skipping classification", "ticker", event.Ticker, "error", err)
		return resp, nil
	}

	// 3. Classify all indicators and persist statuses
	classified := signal.ClassifyAll(tech, latestPrice.Close)

	statusStrings := make(map[string]string, len(classified))
	for k, v := range classified {
		statusStrings[k] = string(v)
	}

	if err := statuses.UpdateIndicatorStatuses(ctx, event.TickerID, date, statusStrings); err != nil {
		return nil, fmt.Errorf("updating indicator statuses for %s: %w", event.Ticker, err)
	}
	resp.IndicatorsClassified = len(statusStrings)

	logger.Info("enriched ticker", "ticker", event.Ticker, "sector", resp.Sector, "indicators", resp.IndicatorsClassified)
	return resp, nil
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadEnrichTicker()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	fundRepo := repository.NewTickerFundamentalsRepo(pool, logger)
	techRepo := repository.NewTickerTechnicalsRepo(pool, logger)
	priceRepo := repository.NewDailyPriceRepo(pool, logger)
	tickerRepo := repository.NewTickerRepo(pool, logger)
	stageRepo := repository.NewPipelineTickerStageRepo(pool, logger)

	return enrichTicker(ctx, event, fundRepo, techRepo, priceRepo, tickerRepo, techRepo, stageRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
