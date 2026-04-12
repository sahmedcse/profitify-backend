package indicator_test

import (
	"math"
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
	"github.com/profitify/profitify-backend/internal/indicator"
)

func TestComputeOBV_Basic(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 105, Volume: 2000}, // up: +2000
		{Close: 103, Volume: 1500}, // down: -1500
		{Close: 108, Volume: 3000}, // up: +3000
	}
	// OBV = 0 + 2000 - 1500 + 3000 = 3500
	obv, err := indicator.ComputeOBV(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(obv-3500) > 0.001 {
		t.Errorf("OBV = %v, want 3500", obv)
	}
}

func TestComputeOBV_FlatClose(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 100, Volume: 2000}, // flat: no change
	}
	obv, err := indicator.ComputeOBV(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obv != 0 {
		t.Errorf("OBV = %v, want 0 for flat closes", obv)
	}
}

func TestComputeOBV_AllDown(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
		{Close: 95, Volume: 2000},
		{Close: 90, Volume: 3000},
	}
	// OBV = -2000 - 3000 = -5000
	obv, err := indicator.ComputeOBV(prices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(obv-(-5000)) > 0.001 {
		t.Errorf("OBV = %v, want -5000", obv)
	}
}

func TestComputeOBV_InsufficientData(t *testing.T) {
	prices := []domain.DailyPrice{
		{Close: 100, Volume: 1000},
	}
	_, err := indicator.ComputeOBV(prices)
	if err == nil {
		t.Fatal("expected error for insufficient data")
	}
}
