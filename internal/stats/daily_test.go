package stats

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestComputeDaily_Basic(t *testing.T) {
	today := domain.DailyPrice{Open: 102, High: 105, Low: 100, Close: 103, Volume: 1200000}
	yesterday := domain.DailyPrice{Close: 100, Volume: 1000000}
	avgVol30d := 1100000.0

	result := ComputeDaily(today, yesterday, avgVol30d)

	if math.Abs(result.PriceChange-3.0) > 0.01 {
		t.Errorf("PriceChange = %.2f, want 3.0", result.PriceChange)
	}
	if math.Abs(result.PriceChangePct-3.0) > 0.01 {
		t.Errorf("PriceChangePct = %.2f, want 3.0", result.PriceChangePct)
	}
	if math.Abs(result.DayRange-5.0) > 0.01 {
		t.Errorf("DayRange = %.2f, want 5.0", result.DayRange)
	}
	if math.Abs(result.GapPct-2.0) > 0.01 {
		t.Errorf("GapPct = %.2f, want 2.0", result.GapPct)
	}
	if math.Abs(result.VolumeChangePct-20.0) > 0.01 {
		t.Errorf("VolumeChangePct = %.2f, want 20.0", result.VolumeChangePct)
	}
	if math.Abs(result.RelativeVolume-1200000.0/1100000.0) > 0.01 {
		t.Errorf("RelativeVolume = %.3f, want ~1.091", result.RelativeVolume)
	}
}

func TestComputeDaily_ZeroYesterday(t *testing.T) {
	today := domain.DailyPrice{Open: 100, High: 105, Low: 95, Close: 100, Volume: 1000}
	yesterday := domain.DailyPrice{}

	result := ComputeDaily(today, yesterday, 0)

	if result.PriceChange != 0 {
		t.Error("expected zero PriceChange when yesterday close is 0")
	}
	if result.DayRange != 10.0 {
		t.Errorf("DayRange = %.2f, want 10.0", result.DayRange)
	}
}
