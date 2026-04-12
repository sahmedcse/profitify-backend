package main

import (
	"context"
	"encoding/json"
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
	"github.com/profitify/profitify-backend/internal/stats"
)

// priceReader abstracts the daily price repository for testing.
type priceReader interface {
	GetByTickerAndDateRange(ctx context.Context, tickerID string, from, to time.Time) ([]domain.DailyPrice, error)
}

// technicalsReader abstracts the technicals repository for testing.
type technicalsReader interface {
	GetLatest(ctx context.Context, tickerID string) (*domain.TechnicalIndicators, error)
}

// statsWriter abstracts the stats repository for testing.
type statsWriter interface {
	Upsert(ctx context.Context, stats *domain.TickerStats) error
}

// Response is the output payload for the ComputeStats Lambda.
type Response struct {
	Ticker         string `json:"ticker"`
	Date           string `json:"date"`
	SignalLabel    string `json:"signal_label"`
	SignalStrength int    `json:"signal_strength"`
}

// computeStats is the core logic.
func computeStats(
	ctx context.Context,
	event pipeline.TickerEvent,
	prices priceReader,
	technicals technicalsReader,
	writer statsWriter,
	logger *slog.Logger,
) (*Response, error) {
	if event.TickerID == "" {
		return nil, fmt.Errorf("ticker_id is required")
	}

	date, err := time.Parse("2006-01-02", event.Date)
	if err != nil {
		return nil, fmt.Errorf("parsing date %q: %w", event.Date, err)
	}

	// 1. Read OHLCV history (365 days for 52-week)
	from := date.AddDate(-1, 0, 0)
	ohlcv, err := prices.GetByTickerAndDateRange(ctx, event.TickerID, from, date)
	if err != nil {
		return nil, fmt.Errorf("reading OHLCV for %s: %w", event.Ticker, err)
	}
	if len(ohlcv) < 2 {
		return nil, fmt.Errorf("insufficient OHLCV data for %s: need >=2 bars, got %d", event.Ticker, len(ohlcv))
	}

	today := ohlcv[len(ohlcv)-1]
	yesterday := ohlcv[len(ohlcv)-2]

	// 2. Compute rolling windows
	var rolling7, rolling30, rolling90 *stats.RollingResult
	if len(ohlcv) >= 7 {
		r := stats.ComputeRolling(ohlcv[len(ohlcv)-7:])
		rolling7 = &r
	}
	if len(ohlcv) >= 30 {
		r := stats.ComputeRolling(ohlcv[len(ohlcv)-30:])
		rolling30 = &r
	}
	if len(ohlcv) >= 90 {
		r := stats.ComputeRolling(ohlcv[len(ohlcv)-90:])
		rolling90 = &r
	}

	// 3. Compute daily (needs 30d avg volume)
	avgVol30d := 0.0
	if rolling30 != nil {
		avgVol30d = rolling30.AvgVolume
	}
	daily := stats.ComputeDaily(today, yesterday, avgVol30d)

	// 4. Compute 52-week
	week52 := stats.Compute52Week(ohlcv, today.Close)

	// 5. Compute pivots from last 20 bars
	last20 := ohlcv
	if len(last20) > 20 {
		last20 = last20[len(last20)-20:]
	}
	pivots := stats.ComputePivots(last20)
	pivotsJSON, _ := json.Marshal(pivots)

	// 6. Read technicals for signal aggregation
	tech, err := technicals.GetLatest(ctx, event.TickerID)
	if err != nil {
		logger.Warn("no technicals for signal, using partial", "ticker", event.Ticker, "error", err)
	}

	// 7. Build partial stats for signal.Aggregate
	partialStats := &domain.TickerStats{}
	if rolling30 != nil {
		partialStats.PriceReturn30d = &rolling30.PriceReturn
	}

	label, strength := signal.Aggregate(tech, partialStats, today.Close)

	// 8. Assemble full stats
	tickerStats := &domain.TickerStats{
		TickerID: event.TickerID,
		Time:     date,

		// Daily
		PriceChange:     daily.PriceChange,
		PriceChangePct:  daily.PriceChangePct,
		VolumeChangePct: daily.VolumeChangePct,
		DayRange:        daily.DayRange,
		GapPct:          daily.GapPct,
		RelativeVolume:  daily.RelativeVolume,

		// 52-week
		High52w:            &week52.High52w,
		Low52w:             &week52.Low52w,
		DistFromHigh52wPct: &week52.DistFromHigh52wPct,
		DistFromLow52wPct:  &week52.DistFromLow52wPct,

		// Signal + pivots
		SignalLabel:    label,
		SignalStrength: strength,
		PivotLevels:    pivotsJSON,
	}

	// Rolling windows
	if rolling7 != nil {
		tickerStats.PriceReturn7d = &rolling7.PriceReturn
		tickerStats.Volatility7d = &rolling7.Volatility
		tickerStats.AvgVolume7d = &rolling7.AvgVolume
		tickerStats.MaxDrawdown7d = &rolling7.MaxDrawdown
	}
	if rolling30 != nil {
		tickerStats.PriceReturn30d = &rolling30.PriceReturn
		tickerStats.Volatility30d = &rolling30.Volatility
		tickerStats.AvgVolume30d = &rolling30.AvgVolume
		tickerStats.MaxDrawdown30d = &rolling30.MaxDrawdown
	}
	if rolling90 != nil {
		tickerStats.PriceReturn90d = &rolling90.PriceReturn
		tickerStats.Volatility90d = &rolling90.Volatility
		tickerStats.AvgVolume90d = &rolling90.AvgVolume
		tickerStats.MaxDrawdown90d = &rolling90.MaxDrawdown
	}

	// 9. Upsert
	logger.Info("upserting stats", "ticker", event.Ticker, "signal", label, "strength", strength)
	if err := writer.Upsert(ctx, tickerStats); err != nil {
		return nil, fmt.Errorf("upserting stats for %s: %w", event.Ticker, err)
	}

	return &Response{
		Ticker:         event.Ticker,
		Date:           event.Date,
		SignalLabel:    label,
		SignalStrength: strength,
	}, nil
}

func handleRequest(ctx context.Context, event pipeline.TickerEvent) (*Response, error) {
	logger := lambdautil.InitLogger()

	cfg, err := config.LoadComputeStats()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	priceRepo := repository.NewDailyPriceRepo(pool, logger)
	techRepo := repository.NewTickerTechnicalsRepo(pool, logger)
	statsRepo := repository.NewTickerStatsRepo(pool, logger)

	return computeStats(ctx, event, priceRepo, techRepo, statsRepo, logger)
}

func main() {
	lambda.Start(handleRequest)
}
