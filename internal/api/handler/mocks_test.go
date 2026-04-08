package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/repository"
)

// contextWith returns a copy of r with the given chi route context attached.
// httptest.NewRequest doesn't run through a router, so URLParams are not
// populated by default — we have to inject them manually.
func contextWith(r *http.Request, rctx *chi.Context) context.Context {
	return context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
}

// Shared logger that drops everything.
var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeTickerRepo is a minimal in-memory TickerRepository.
type fakeTickerRepo struct {
	bySymbol  map[string]*domain.Ticker
	dashboard []repository.DashboardTicker
	listErr   error
	getErr    error
}

func (f *fakeTickerRepo) UpsertBatch(_ context.Context, _ []domain.Ticker) error {
	return nil
}
func (f *fakeTickerRepo) GetActive(_ context.Context) ([]domain.Ticker, error) {
	out := []domain.Ticker{}
	for _, t := range f.bySymbol {
		out = append(out, *t)
	}
	return out, nil
}
func (f *fakeTickerRepo) GetBySymbol(_ context.Context, sym string) (*domain.Ticker, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.bySymbol[sym]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return t, nil
}
func (f *fakeTickerRepo) ListForDashboard(_ context.Context, filter repository.DashboardFilter) ([]repository.DashboardTicker, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := f.dashboard
	if filter.Sector != "" {
		filtered := []repository.DashboardTicker{}
		for _, d := range out {
			if d.Sector == filter.Sector {
				filtered = append(filtered, d)
			}
		}
		out = filtered
	}
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

type fakePriceRepo struct {
	bars map[string][]domain.DailyPrice
	err  error
}

func (f *fakePriceRepo) GetRange(_ context.Context, tickerID string, _, _ time.Time) ([]domain.DailyPrice, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.bars[tickerID], nil
}

type fakeTechnicalsRepo struct {
	rows map[string]*domain.TechnicalIndicators
	err  error
}

func (f *fakeTechnicalsRepo) GetLatest(_ context.Context, tickerID string) (*domain.TechnicalIndicators, error) {
	if f.err != nil {
		return nil, f.err
	}
	t, ok := f.rows[tickerID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return t, nil
}

type fakeFundamentalsRepo struct {
	rows map[string]*domain.TickerFundamentals
}

func (f *fakeFundamentalsRepo) GetLatest(_ context.Context, tickerID string) (*domain.TickerFundamentals, error) {
	r, ok := f.rows[tickerID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return r, nil
}

type fakeStatsRepo struct {
	rows map[string]*domain.TickerStats
}

func (f *fakeStatsRepo) GetLatest(_ context.Context, tickerID string) (*domain.TickerStats, error) {
	r, ok := f.rows[tickerID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return r, nil
}

type fakeDividendRepo struct {
	rows map[string]*domain.TickerDividendSummary
}

func (f *fakeDividendRepo) GetLatest(_ context.Context, tickerID string) (*domain.TickerDividendSummary, error) {
	r, ok := f.rows[tickerID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return r, nil
}

func ptr(f float64) *float64 { return &f }

func decodeJSON(t interface {
	Fatalf(format string, args ...any)
}, body io.Reader, into any) {
	if err := json.NewDecoder(body).Decode(into); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
