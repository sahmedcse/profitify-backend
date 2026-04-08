package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/stats"
)

func TestGetLevels(t *testing.T) {
	h, tr, sr, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	payload := stats.PivotLevels{
		R3: stats.PivotLevel{Price: 210.5, Strength: stats.StrengthWeak},
		R2: stats.PivotLevel{Price: 205.0, Strength: stats.StrengthModerate},
		R1: stats.PivotLevel{Price: 200.0, Strength: stats.StrengthStrong},
		S1: stats.PivotLevel{Price: 190.0, Strength: stats.StrengthStrong},
		S2: stats.PivotLevel{Price: 185.0, Strength: stats.StrengthModerate},
		S3: stats.PivotLevel{Price: 180.0, Strength: stats.StrengthWeak},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sr.rows["id1"] = &domain.TickerStats{PivotLevels: raw}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/levels", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetLevels(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp struct {
		Symbol string            `json:"symbol"`
		Levels stats.PivotLevels `json:"levels"`
	}
	decodeJSON(t, w.Body, &resp)
	if resp.Symbol != "AAPL" {
		t.Errorf("symbol = %q, want AAPL", resp.Symbol)
	}
	if resp.Levels.R1.Price != 200 {
		t.Errorf("R1 = %v, want 200", resp.Levels.R1.Price)
	}
	if resp.Levels.R1.Strength != stats.StrengthStrong {
		t.Errorf("R1 strength = %q, want %q", resp.Levels.R1.Strength, stats.StrengthStrong)
	}
	if resp.Levels.S3.Price != 180 {
		t.Errorf("S3 = %v, want 180", resp.Levels.S3.Price)
	}
}

// When pivot_levels has never been materialized for a ticker, the
// stored RawMessage is empty. The handler should return 200 with the
// zero-value PivotLevels struct rather than crashing on json.Unmarshal.
func TestGetLevels_EmptyPayload(t *testing.T) {
	h, tr, sr, _, _, _, _ := newTestHandlers(t)
	tr.bySymbol["AAPL"] = &domain.Ticker{ID: "id1", Ticker: "AAPL"}
	sr.rows["id1"] = &domain.TickerStats{}

	r := httptest.NewRequest("GET", "/v1/tickers/AAPL/levels", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "AAPL")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetLevels(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestGetLevels_TickerNotFound(t *testing.T) {
	h, _, _, _, _, _, _ := newTestHandlers(t)
	r := httptest.NewRequest("GET", "/v1/tickers/NOPE/levels", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("symbol", "NOPE")
	r = r.WithContext(contextWith(r, rctx))
	w := httptest.NewRecorder()
	h.GetLevels(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}
