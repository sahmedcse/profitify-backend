package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/profitify/profitify-backend/internal/signal"
)

// indicatorRow is one entry in the dashboard's Technical Indicators panel.
type indicatorRow struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Status string  `json:"status"`
	Detail string  `json:"detail"`
}

// indicatorsResponse is the wrapper sent for /technicals.
type indicatorsResponse struct {
	Symbol     string         `json:"symbol"`
	Indicators []indicatorRow `json:"indicators"`
	BuyCount   int            `json:"buy_count"`
	NeutralCnt int            `json:"neutral_count"`
	SellCount  int            `json:"sell_count"`
}

// GetTechnicals handles GET /v1/tickers/{symbol}/technicals.
//
// Returns the latest technical indicator snapshot, classified into
// bullish/neutral/bearish per indicator with a human-readable detail
// string, plus the buy/neutral/sell counter the dashboard shows next
// to the panel header.
//
// The classifier output is read from the materialized
// ticker_technicals.indicator_statuses JSONB column when present, and
// computed on the fly as a fallback so this endpoint works against
// historical rows that predate the materialization pipeline.
func (h *Handlers) GetTechnicals(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	if symbol == "" {
		h.writeError(w, http.StatusBadRequest, "symbol required")
		return
	}
	t, err := h.Tickers.GetBySymbol(r.Context(), symbol)
	if err != nil {
		h.notFoundOrInternal(w, err, "ticker")
		return
	}
	tech, err := h.Technicals.GetLatest(r.Context(), t.ID)
	if err != nil {
		h.notFoundOrInternal(w, err, "technicals")
		return
	}

	// We need the latest close for the fallback classifier path and to
	// render SMA/EMA detail strings. Look two days either side of the
	// technicals timestamp so we always pick something up regardless of
	// whether the bar landed on the same day.
	var closePrice float64
	bars, _ := h.Prices.GetRange(r.Context(), t.ID,
		tech.Time.Add(-2*24*time.Hour), tech.Time.Add(2*24*time.Hour))
	if len(bars) > 0 {
		closePrice = bars[len(bars)-1].Close
	}

	statuses := tech.IndicatorStatuses
	if len(statuses) == 0 {
		statuses = map[string]string{}
		for k, v := range signal.ClassifyAll(tech, closePrice) {
			statuses[k] = string(v)
		}
	}

	rows := []indicatorRow{}
	addRow := func(name string, value *float64) {
		if value == nil {
			return
		}
		status := signal.Status(statuses[name])
		rows = append(rows, indicatorRow{
			Name:   name,
			Value:  *value,
			Status: string(status),
			Detail: signal.DetailFor(name, *value, status),
		})
	}
	addRow(signal.IndicatorRSI, tech.RSI14)
	addRow(signal.IndicatorMACD, tech.MACDLine)
	addRow(signal.IndicatorSMA20, tech.SMA20)
	addRow(signal.IndicatorSMA50, tech.SMA50)
	addRow(signal.IndicatorEMA12, tech.EMA12)
	if tech.BollingerMiddle != nil {
		addRow(signal.IndicatorBollinger, tech.BollingerMiddle)
	}

	resp := indicatorsResponse{Symbol: t.Ticker, Indicators: rows}
	for _, row := range rows {
		switch signal.Status(row.Status) {
		case signal.StatusBullish:
			resp.BuyCount++
		case signal.StatusBearish:
			resp.SellCount++
		default:
			resp.NeutralCnt++
		}
	}
	h.writeJSON(w, http.StatusOK, resp)
}
