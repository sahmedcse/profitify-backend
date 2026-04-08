package stats_test

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/stats"
)

func TestNormalizeSeries(t *testing.T) {
	tests := []struct {
		name   string
		prices []domain.DailyPrice
		want   []float64
	}{
		{
			name:   "empty",
			prices: nil,
			want:   nil,
		},
		{
			name: "zero base returns nil",
			prices: []domain.DailyPrice{
				{Close: 0}, {Close: 100},
			},
			want: nil,
		},
		{
			name: "single element",
			prices: []domain.DailyPrice{
				{Close: 100},
			},
			want: []float64{0},
		},
		{
			name: "simple ascending",
			prices: []domain.DailyPrice{
				{Close: 100}, {Close: 110}, {Close: 120}, {Close: 95},
			},
			want: []float64{0, 10, 20, -5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stats.NormalizeSeries(tt.prices)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if math.Abs(got[i]-tt.want[i]) > 1e-9 {
					t.Errorf("[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
