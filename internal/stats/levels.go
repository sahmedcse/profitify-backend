// Package stats contains pure numerical helpers used by both the
// dashboard API and the materialization pipeline. Functions here take
// slices of domain types and return primitive results — no I/O, no
// global state, no logging.
package stats

import "github.com/profitify/profitify-backend/internal/domain"

// Strength tier for a pivot level. Reflects how many days in the
// reference window closed within touch_threshold of the level.
const (
	StrengthStrong   = "strong"
	StrengthModerate = "moderate"
	StrengthWeak     = "weak"
)

// PivotLevel is one resistance or support price + how often historical
// closes touched it.
type PivotLevel struct {
	Price    float64 `json:"price"`
	Strength string  `json:"strength"`
}

// PivotLevels groups the six classic pivot points calculated from the
// most recent OHLC bar in a window. R = resistance (above), S = support
// (below). The order R3 → R1 → S1 → S3 reflects descending price.
type PivotLevels struct {
	R3 PivotLevel `json:"R3"`
	R2 PivotLevel `json:"R2"`
	R1 PivotLevel `json:"R1"`
	S1 PivotLevel `json:"S1"`
	S2 PivotLevel `json:"S2"`
	S3 PivotLevel `json:"S3"`
}

// touchThresholdPct is how close (as a fraction of price) a daily close
// must be to a pivot level to count as a "touch" when classifying strength.
const touchThresholdPct = 0.005 // 0.5%

// ComputePivots derives classic pivot points from the most recent bar in
// `recent` (it must be the last element) and classifies the strength of
// each level by how many bars in the entire `recent` slice closed within
// touchThresholdPct of that level.
//
// Returns a zero-value PivotLevels if `recent` is empty. With a single
// bar the strengths will all be "weak" because there's no history to
// confirm them.
//
// Classic 5-point pivot formula:
//
//	P  = (H + L + C) / 3
//	R1 = 2P − L              S1 = 2P − H
//	R2 = P + (H − L)         S2 = P − (H − L)
//	R3 = H + 2(P − L)        S3 = L − 2(H − P)
func ComputePivots(recent []domain.DailyPrice) PivotLevels {
	if len(recent) == 0 {
		return PivotLevels{}
	}
	last := recent[len(recent)-1]
	h, l, c := last.High, last.Low, last.Close
	p := (h + l + c) / 3
	rng := h - l

	r1 := 2*p - l
	s1 := 2*p - h
	r2 := p + rng
	s2 := p - rng
	r3 := h + 2*(p-l)
	s3 := l - 2*(h-p)

	return PivotLevels{
		R3: PivotLevel{Price: r3, Strength: classifyStrength(recent, r3)},
		R2: PivotLevel{Price: r2, Strength: classifyStrength(recent, r2)},
		R1: PivotLevel{Price: r1, Strength: classifyStrength(recent, r1)},
		S1: PivotLevel{Price: s1, Strength: classifyStrength(recent, s1)},
		S2: PivotLevel{Price: s2, Strength: classifyStrength(recent, s2)},
		S3: PivotLevel{Price: s3, Strength: classifyStrength(recent, s3)},
	}
}

// classifyStrength buckets a level by how many recent bars closed within
// touchThresholdPct of it.
//
//	≥3 touches → strong
//	1-2 touches → moderate
//	0 touches → weak
func classifyStrength(recent []domain.DailyPrice, level float64) string {
	if level <= 0 {
		return StrengthWeak
	}
	tol := level * touchThresholdPct
	touches := 0
	for _, bar := range recent {
		if abs(bar.Close-level) <= tol {
			touches++
		}
	}
	switch {
	case touches >= 3:
		return StrengthStrong
	case touches >= 1:
		return StrengthModerate
	default:
		return StrengthWeak
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
