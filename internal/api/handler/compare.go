package handler

import (
	"net/http"
	"time"

	"github.com/profitify/profitify-backend/internal/stats"
)

// comparePoint is one weekly sample of the normalized comparison series.
// The two ticker pct values are stored under the symbol-keyed fields the
// frontend's Recharts uses directly.
type comparePoint struct {
	Time time.Time `json:"time"`
	A    float64   `json:"a_pct"`
	B    float64   `json:"b_pct"`
}

type compareResponse struct {
	A         string         `json:"a"`
	B         string         `json:"b"`
	Period    string         `json:"period"`
	Series    []comparePoint `json:"series"`
	SummaryA  float64        `json:"summary_a_pct"`
	SummaryB  float64        `json:"summary_b_pct"`
}

// GetCompare handles GET /v1/compare?a=…&b=…&period=3M|6M|1Y.
//
// Returns two normalized return series, week-sampled, plus the final
// percentage change for each ticker. The data is computed on the fly
// from the daily_prices hypertable — there's no point materializing
// every (a, b) pair, that would be O(n²).
func (h *Handlers) GetCompare(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	aSym, bSym := q.Get("a"), q.Get("b")
	if aSym == "" || bSym == "" {
		h.writeError(w, http.StatusBadRequest, "a and b required")
		return
	}
	window, ok := timeframeToDuration(q.Get("period"))
	if !ok {
		h.writeError(w, http.StatusBadRequest, "invalid period")
		return
	}

	aTicker, err := h.Tickers.GetBySymbol(r.Context(), aSym)
	if err != nil {
		h.notFoundOrInternal(w, err, "ticker a")
		return
	}
	bTicker, err := h.Tickers.GetBySymbol(r.Context(), bSym)
	if err != nil {
		h.notFoundOrInternal(w, err, "ticker b")
		return
	}

	to := time.Now().UTC()
	from := to.Add(-window)
	aBars, err := h.Prices.GetRange(r.Context(), aTicker.ID, from, to)
	if err != nil {
		h.Logger.Error("compare aBars", "error", err.Error())
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	bBars, err := h.Prices.GetRange(r.Context(), bTicker.ID, from, to)
	if err != nil {
		h.Logger.Error("compare bBars", "error", err.Error())
		h.writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	aSeries := stats.NormalizeSeries(aBars)
	bSeries := stats.NormalizeSeries(bBars)

	// Sample weekly so the chart isn't dominated by daily noise. Pick
	// every 5th element from the longer series and align by index against
	// the shorter one — both have the same trading-day cadence.
	const step = 5
	out := []comparePoint{}
	n := minInt(len(aSeries), len(bSeries))
	for i := 0; i < n; i += step {
		out = append(out, comparePoint{
			Time: aBars[i].Time,
			A:    aSeries[i],
			B:    bSeries[i],
		})
	}

	resp := compareResponse{
		A: aTicker.Ticker, B: bTicker.Ticker,
		Period: q.Get("period"), Series: out,
	}
	if len(aSeries) > 0 {
		resp.SummaryA = aSeries[len(aSeries)-1]
	}
	if len(bSeries) > 0 {
		resp.SummaryB = bSeries[len(bSeries)-1]
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
