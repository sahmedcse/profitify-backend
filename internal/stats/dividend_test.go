package stats

import (
	"testing"
	"time"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestComputeDividendSummary_HappyPath(t *testing.T) {
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	dividends := []domain.TickerDividend{
		{CashAmount: 0.25, ExDividendDate: "2026-03-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.25, ExDividendDate: "2025-12-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.25, ExDividendDate: "2025-09-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.25, ExDividendDate: "2025-06-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.24, ExDividendDate: "2025-03-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.24, ExDividendDate: "2024-12-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.24, ExDividendDate: "2024-09-15", Frequency: 4, DistributionType: "CD"},
		{CashAmount: 0.24, ExDividendDate: "2024-06-15", Frequency: 4, DistributionType: "CD"},
	}

	summary := ComputeDividendSummary(dividends, 175.0, asOf)

	// Current yield: 0.25 * 4 / 175 * 100 ≈ 0.571%
	if summary.CurrentYield == nil {
		t.Fatal("expected CurrentYield to be set")
	}
	if *summary.CurrentYield < 0.5 || *summary.CurrentYield > 0.6 {
		t.Errorf("CurrentYield = %.3f, want ~0.571", *summary.CurrentYield)
	}

	if summary.TrailingYield12m == nil {
		t.Fatal("expected TrailingYield12m to be set")
	}

	if summary.PayoutFrequency != 4 {
		t.Errorf("PayoutFrequency = %d, want 4", summary.PayoutFrequency)
	}
	if summary.LatestDistributionType != "CD" {
		t.Errorf("LatestDistributionType = %q, want CD", summary.LatestDistributionType)
	}
	if summary.NextExDividendDate == nil {
		t.Fatal("expected NextExDividendDate to be set")
	}
	if summary.DaysUntilExDividend == nil {
		t.Fatal("expected DaysUntilExDividend to be set")
	}
}

func TestComputeDividendSummary_Empty(t *testing.T) {
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	summary := ComputeDividendSummary(nil, 175.0, asOf)

	if summary.CurrentYield != nil {
		t.Error("expected nil CurrentYield for empty dividends")
	}
}

func TestComputeDividendSummary_ZeroPrice(t *testing.T) {
	dividends := []domain.TickerDividend{
		{CashAmount: 0.25, ExDividendDate: "2026-03-15", Frequency: 4},
	}
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	summary := ComputeDividendSummary(dividends, 0, asOf)

	if summary.CurrentYield != nil {
		t.Error("expected nil CurrentYield when price is zero")
	}
}

func TestComputeDividendSummary_ConsecutiveIncreases(t *testing.T) {
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	dividends := []domain.TickerDividend{
		// 2025: total = 1.00 (4 × 0.25)
		{CashAmount: 0.25, ExDividendDate: "2025-12-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-09-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-06-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-03-15", Frequency: 4},
		// 2024: total = 0.96 (4 × 0.24)
		{CashAmount: 0.24, ExDividendDate: "2024-12-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-09-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-06-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-03-15", Frequency: 4},
		// 2023: total = 0.92 (4 × 0.23)
		{CashAmount: 0.23, ExDividendDate: "2023-12-15", Frequency: 4},
		{CashAmount: 0.23, ExDividendDate: "2023-09-15", Frequency: 4},
		{CashAmount: 0.23, ExDividendDate: "2023-06-15", Frequency: 4},
		{CashAmount: 0.23, ExDividendDate: "2023-03-15", Frequency: 4},
	}

	summary := ComputeDividendSummary(dividends, 175.0, asOf)

	if summary.ConsecutiveIncreases != 2 { // 2025 > 2024 > 2023
		t.Errorf("ConsecutiveIncreases = %d, want 2", summary.ConsecutiveIncreases)
	}
}

func TestComputeDividendSummary_GrowthRates(t *testing.T) {
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	dividends := []domain.TickerDividend{
		// 2025: total = 1.00
		{CashAmount: 0.25, ExDividendDate: "2025-12-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-09-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-06-15", Frequency: 4},
		{CashAmount: 0.25, ExDividendDate: "2025-03-15", Frequency: 4},
		// 2024: total = 0.96
		{CashAmount: 0.24, ExDividendDate: "2024-12-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-09-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-06-15", Frequency: 4},
		{CashAmount: 0.24, ExDividendDate: "2024-03-15", Frequency: 4},
	}

	summary := ComputeDividendSummary(dividends, 175.0, asOf)

	// 1y growth: (1.00 / 0.96 - 1) × 100 ≈ 4.17%
	if summary.DividendGrowthRate1y == nil {
		t.Fatal("expected DividendGrowthRate1y to be set")
	}
	rate := *summary.DividendGrowthRate1y
	if rate < 4.0 || rate > 4.5 {
		t.Errorf("DividendGrowthRate1y = %.2f, want ~4.17", rate)
	}
}

func TestComputeDividendSummary_NextExDividendDate(t *testing.T) {
	asOf := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	dividends := []domain.TickerDividend{
		{CashAmount: 0.25, ExDividendDate: "2026-03-15", Frequency: 4}, // quarterly → next ~June 15
	}

	summary := ComputeDividendSummary(dividends, 175.0, asOf)

	if summary.NextExDividendDate == nil {
		t.Fatal("expected NextExDividendDate to be set")
	}
	// 365/4 ≈ 91 days after 2026-03-15 = ~2026-06-14
	if *summary.NextExDividendDate < "2026-06" {
		t.Errorf("NextExDividendDate = %q, expected ~2026-06-14", *summary.NextExDividendDate)
	}
	if summary.DaysUntilExDividend == nil || *summary.DaysUntilExDividend < 0 {
		t.Error("expected positive DaysUntilExDividend")
	}
}
