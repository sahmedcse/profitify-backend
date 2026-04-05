package domain

import "time"

// TechnicalIndicators holds technical analysis indicators for a ticker on a given date.
// Massive-sourced: SMA, EMA, RSI, MACD (split-adjusted).
// Self-computed: Bollinger Bands, ATR, OBV.
type TechnicalIndicators struct {
	ID     string    `json:"id"`
	TickerID string  `json:"ticker_id"`
	Time   time.Time `json:"time"`

	// Simple Moving Averages (Massive)
	SMA20  *float64 `json:"sma_20"`
	SMA50  *float64 `json:"sma_50"`
	SMA200 *float64 `json:"sma_200"`

	// Exponential Moving Averages (Massive)
	EMA12 *float64 `json:"ema_12"`
	EMA26 *float64 `json:"ema_26"`

	// Relative Strength Index (Massive)
	RSI14 *float64 `json:"rsi_14"`

	// MACD (Massive)
	MACDLine      *float64 `json:"macd_line"`
	MACDSignal    *float64 `json:"macd_signal"`
	MACDHistogram *float64 `json:"macd_histogram"`

	// Bollinger Bands (self-computed, 20-period, 2 std dev)
	BollingerUpper  *float64 `json:"bollinger_upper"`
	BollingerMiddle *float64 `json:"bollinger_middle"`
	BollingerLower  *float64 `json:"bollinger_lower"`

	// Average True Range (self-computed, 14-period)
	ATR14 *float64 `json:"atr_14"`

	// On-Balance Volume (self-computed)
	OBV *float64 `json:"obv"`
}
