package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// timeframeToDuration maps the dashboard's timeframe param to a lookback
// window. 1D is intentionally absent — the daily_prices hypertable is
// daily-granularity so the smallest meaningful window is 1M.
func timeframeToDuration(tf string) (time.Duration, bool) {
	switch tf {
	case "", "1M":
		return 30 * 24 * time.Hour, true
	case "3M":
		return 90 * 24 * time.Hour, true
	case "6M":
		return 180 * 24 * time.Hour, true
	case "1Y":
		return 365 * 24 * time.Hour, true
	case "ALL":
		return 20 * 365 * 24 * time.Hour, true
	}
	return 0, false
}

// pricePoint is the slim OHLCV shape sent to the dashboard. We strip the
// internal id and ticker_id from the response since the route already
// scopes them.
type pricePoint struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// GetPrices handles GET /v1/tickers/{symbol}/prices?timeframe=…
// Returns OHLCV bars over the requested window in chronological order.
func (h *Handlers) GetPrices(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		h.writeError(w, http.StatusBadRequest, "symbol required")
		return
	}
	window, ok := timeframeToDuration(r.URL.Query().Get("timeframe"))
	if !ok {
		h.writeError(w, http.StatusBadRequest, "invalid timeframe")
		return
	}

	t, err := h.Tickers.GetBySymbol(r.Context(), symbol)
	if err != nil {
		h.notFoundOrInternal(w, err, "ticker")
		return
	}

	to := time.Now().UTC()
	from := to.Add(-window)
	bars, err := h.Prices.GetRange(r.Context(), t.ID, from, to)
	if err != nil {
		h.Logger.Error("GetRange", "error", err.Error())
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	out := make([]pricePoint, 0, len(bars))
	for _, b := range bars {
		out = append(out, pricePoint{
			Time: b.Time, Open: b.Open, High: b.High,
			Low: b.Low, Close: b.Close, Volume: b.Volume,
		})
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"symbol":    t.Ticker,
		"timeframe": r.URL.Query().Get("timeframe"),
		"bars":      out,
	})
}
