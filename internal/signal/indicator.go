// Package signal contains pure classifier and aggregator functions that
// turn raw technical indicator values into the bullish/neutral/bearish
// statuses and human-readable detail strings shown in the dashboard UI.
//
// All functions are deterministic and side-effect free so they can be
// called from the materialization pipeline (enrich-ticker / compute-stats
// Lambdas) and from tests with no infrastructure.
package signal

import "fmt"

// Status is the classification of a single indicator's reading.
type Status string

const (
	StatusBullish Status = "bullish"
	StatusNeutral Status = "neutral"
	StatusBearish Status = "bearish"
)

// Indicator names used as JSON keys in ticker_technicals.indicator_statuses
// and as the first argument to DetailFor.
const (
	IndicatorRSI       = "rsi_14"
	IndicatorMACD      = "macd"
	IndicatorSMA20     = "sma_20"
	IndicatorSMA50     = "sma_50"
	IndicatorEMA12     = "ema_12"
	IndicatorBollinger = "bollinger"
)

// ClassifyRSI classifies a 14-period RSI value.
//
// Convention:
//   - <30  oversold  -> bearish (price has dropped, may bounce but trend is down)
//   - >70  overbought -> bullish (price has risen strongly)
//   - 30-70           -> neutral
func ClassifyRSI(rsi float64) Status {
	switch {
	case rsi < 30:
		return StatusBearish
	case rsi > 70:
		return StatusBullish
	default:
		return StatusNeutral
	}
}

// ClassifyMACD classifies the MACD line vs its signal line.
// Line above signal = bullish crossover, below = bearish.
func ClassifyMACD(line, signal float64) Status {
	switch {
	case line > signal:
		return StatusBullish
	case line < signal:
		return StatusBearish
	default:
		return StatusNeutral
	}
}

// ClassifySMA classifies the latest close vs a simple moving average.
// Close above MA = price trending up = bullish.
func ClassifySMA(close, sma float64) Status {
	return classifyVsMA(close, sma)
}

// ClassifyEMA classifies the latest close vs an exponential moving average.
func ClassifyEMA(close, ema float64) Status {
	return classifyVsMA(close, ema)
}

func classifyVsMA(close, ma float64) Status {
	switch {
	case close > ma:
		return StatusBullish
	case close < ma:
		return StatusBearish
	default:
		return StatusNeutral
	}
}

// ClassifyBollinger classifies the latest close against Bollinger Bands.
// Close near upper band -> bearish (overbought, mean-reversion downward bias).
// Close near lower band -> bullish (oversold, mean-reversion upward bias).
func ClassifyBollinger(close, upper, _, lower float64) Status {
	if upper <= lower {
		return StatusNeutral
	}
	width := upper - lower
	// "Near" = within 10% of band width from each edge.
	threshold := width * 0.10
	switch {
	case close >= upper-threshold:
		return StatusBearish
	case close <= lower+threshold:
		return StatusBullish
	default:
		return StatusNeutral
	}
}

// DetailFor renders a short human-readable description for an indicator
// row in the dashboard's Technical Indicators panel.
func DetailFor(indicator string, value float64, status Status) string {
	switch indicator {
	case IndicatorRSI:
		switch status {
		case StatusBearish:
			return fmt.Sprintf("Oversold at %.1f", value)
		case StatusBullish:
			return fmt.Sprintf("Overbought at %.1f", value)
		default:
			if value >= 60 {
				return fmt.Sprintf("Approaching overbought at %.1f", value)
			}
			if value <= 40 {
				return fmt.Sprintf("Approaching oversold at %.1f", value)
			}
			return fmt.Sprintf("Neutral at %.1f", value)
		}
	case IndicatorMACD:
		switch status {
		case StatusBullish:
			return "MACD above signal line"
		case StatusBearish:
			return "MACD below signal line"
		default:
			return "MACD crossing signal line"
		}
	case IndicatorSMA20, IndicatorSMA50, IndicatorEMA12:
		switch status {
		case StatusBullish:
			return fmt.Sprintf("Price above MA at %.2f", value)
		case StatusBearish:
			return fmt.Sprintf("Price below MA at %.2f", value)
		default:
			return fmt.Sprintf("Price at MA %.2f", value)
		}
	case IndicatorBollinger:
		switch status {
		case StatusBullish:
			return "Near lower band"
		case StatusBearish:
			return "Near upper band"
		default:
			return "Inside bands"
		}
	}
	return ""
}
