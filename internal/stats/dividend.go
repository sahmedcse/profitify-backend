package stats

import (
	"math"
	"sort"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
)

// ComputeDividendSummary computes dividend yield, growth, and frequency metrics
// from raw dividend history, the latest stock price, and a reference date.
func ComputeDividendSummary(dividends []domain.TickerDividend, latestClose float64, asOf time.Time) *domain.TickerDividendSummary {
	if len(dividends) == 0 || latestClose <= 0 {
		return &domain.TickerDividendSummary{}
	}

	// Sort by ex_dividend_date descending (most recent first).
	sorted := make([]domain.TickerDividend, len(dividends))
	copy(sorted, dividends)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ExDividendDate > sorted[j].ExDividendDate
	})

	latest := sorted[0]

	summary := &domain.TickerDividendSummary{
		PayoutFrequency:        latest.Frequency,
		LatestDistributionType: latest.DistributionType,
	}

	// Trailing 12-month yield: sum dividends in [asOf-12m, asOf] / close × 100.
	oneYearAgo := asOf.AddDate(-1, 0, 0)
	var trailing12m float64
	for _, d := range sorted {
		dt, err := time.Parse("2006-01-02", d.ExDividendDate)
		if err != nil {
			continue
		}
		if dt.After(oneYearAgo) && !dt.After(asOf) {
			trailing12m += d.CashAmount
		}
	}
	if trailing12m > 0 {
		ty := (trailing12m / latestClose) * 100
		summary.TrailingYield12m = &ty
	}

	// Current / forward yield: latest dividend annualised.
	if latest.Frequency > 0 {
		annualised := latest.CashAmount * float64(latest.Frequency)
		cy := (annualised / latestClose) * 100
		summary.CurrentYield = &cy
		summary.ForwardYield = &cy
	}

	// Group dividends by calendar year for growth analysis.
	yearTotals := make(map[int]float64)
	for _, d := range sorted {
		dt, err := time.Parse("2006-01-02", d.ExDividendDate)
		if err != nil {
			continue
		}
		yearTotals[dt.Year()] += d.CashAmount
	}

	currentYear := asOf.Year()

	// 1-year growth rate.
	if y0, ok := yearTotals[currentYear-1]; ok {
		if y1, ok := yearTotals[currentYear-2]; ok && y1 > 0 {
			rate := ((y0 / y1) - 1) * 100
			summary.DividendGrowthRate1y = &rate
		}
	}

	// 3-year CAGR.
	if y0, ok := yearTotals[currentYear-1]; ok {
		if y3, ok := yearTotals[currentYear-4]; ok && y3 > 0 {
			cagr := (math.Pow(y0/y3, 1.0/3.0) - 1) * 100
			summary.DividendGrowthRate3y = &cagr
		}
	}

	// 5-year CAGR.
	if y0, ok := yearTotals[currentYear-1]; ok {
		if y5, ok := yearTotals[currentYear-6]; ok && y5 > 0 {
			cagr := (math.Pow(y0/y5, 1.0/5.0) - 1) * 100
			summary.DividendGrowthRate5y = &cagr
		}
	}

	// Consecutive annual increases (most recent years going backwards).
	years := make([]int, 0, len(yearTotals))
	for y := range yearTotals {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	consecutive := 0
	for i := 0; i < len(years)-1; i++ {
		if years[i]-years[i+1] != 1 {
			break
		}
		if yearTotals[years[i]] > yearTotals[years[i+1]] {
			consecutive++
		} else {
			break
		}
	}
	summary.ConsecutiveIncreases = consecutive

	// Extrapolate next ex-dividend date from latest + frequency.
	if latest.Frequency > 0 {
		dt, err := time.Parse("2006-01-02", latest.ExDividendDate)
		if err == nil {
			daysPerPeriod := 365 / latest.Frequency
			next := dt.AddDate(0, 0, daysPerPeriod)
			for next.Before(asOf) {
				next = next.AddDate(0, 0, daysPerPeriod)
			}
			nextStr := next.Format("2006-01-02")
			summary.NextExDividendDate = &nextStr
			days := int(next.Sub(asOf).Hours() / 24)
			summary.DaysUntilExDividend = &days
		}
	}

	return summary
}
