package signal

import "github.com/profitify/profitify-backend/internal/domain"

// Aggregate combines a snapshot of technical indicators and rolling stats
// into a single dashboard-facing label and 0-100 strength score.
//
// The function is intentionally tolerant: any missing input (nil pointers)
// is skipped rather than treated as zero. This lets it run against partially
// populated rows during early backfill.
//
// Returned label values:
//   - "Strong Buy"  — strength ≥ 80
//   - "Bullish"     — 60 ≤ strength < 80
//   - "Neutral"     — 40 ≤ strength < 60
//   - "Bearish"     — 20 ≤ strength < 40
//   - "Strong Sell" — strength < 20
//
// Strength is the bullish share of classified indicators (0-100), nudged
// by the 30-day price return so a strongly-trending name is rewarded even
// when its individual indicators are mid-range.
//
// `closePrice` is the latest daily close — needed because SMA/EMA/Bollinger
// classification is relative to price, not just the MA value itself.
func Aggregate(tech *domain.TechnicalIndicators, stats *domain.TickerStats, closePrice float64) (string, int) {
	statuses := ClassifyAll(tech, closePrice)

	var bullish, neutral, total int
	for _, s := range statuses {
		switch s {
		case StatusBullish:
			bullish++
			total++
		case StatusBearish:
			total++
		case StatusNeutral:
			neutral++
			total++
		}
	}

	// Neutral readings count as half-bullish so an all-neutral snapshot maps
	// to a 50 strength score (the midpoint label "Neutral") instead of 0.
	var strength float64
	if total > 0 {
		strength = ((float64(bullish) + 0.5*float64(neutral)) / float64(total)) * 100
	} else {
		strength = 50
	}

	// Nudge by 30-day return: ±1% return ≈ ±1 strength point, capped ±15.
	if stats != nil && stats.PriceReturn30d != nil {
		nudge := *stats.PriceReturn30d
		if nudge > 15 {
			nudge = 15
		} else if nudge < -15 {
			nudge = -15
		}
		strength += nudge
	}

	if strength < 0 {
		strength = 0
	} else if strength > 100 {
		strength = 100
	}

	score := int(strength + 0.5)
	return labelFor(score), score
}

// ClassifyAll runs every classifier we know how to run against the given
// indicator snapshot and returns a map keyed by indicator name. The returned
// map is suitable for direct serialization into ticker_technicals.indicator_statuses.
func ClassifyAll(tech *domain.TechnicalIndicators, closePrice float64) map[string]Status {
	out := map[string]Status{}
	if tech == nil {
		return out
	}
	if tech.RSI14 != nil {
		out[IndicatorRSI] = ClassifyRSI(*tech.RSI14)
	}
	if tech.MACDLine != nil && tech.MACDSignal != nil {
		out[IndicatorMACD] = ClassifyMACD(*tech.MACDLine, *tech.MACDSignal)
	}
	if tech.SMA20 != nil && closePrice > 0 {
		out[IndicatorSMA20] = ClassifySMA(closePrice, *tech.SMA20)
	}
	if tech.SMA50 != nil && closePrice > 0 {
		out[IndicatorSMA50] = ClassifySMA(closePrice, *tech.SMA50)
	}
	if tech.EMA12 != nil && closePrice > 0 {
		out[IndicatorEMA12] = ClassifyEMA(closePrice, *tech.EMA12)
	}
	if tech.BollingerUpper != nil && tech.BollingerMiddle != nil && tech.BollingerLower != nil && closePrice > 0 {
		out[IndicatorBollinger] = ClassifyBollinger(closePrice, *tech.BollingerUpper, *tech.BollingerMiddle, *tech.BollingerLower)
	}
	return out
}

func labelFor(score int) string {
	switch {
	case score >= 80:
		return "Strong Buy"
	case score >= 60:
		return "Bullish"
	case score >= 40:
		return "Neutral"
	case score >= 20:
		return "Bearish"
	default:
		return "Strong Sell"
	}
}
