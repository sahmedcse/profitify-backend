package signal_test

import (
	"strings"
	"testing"

	"github.com/profitify/profitify-backend/internal/signal"
)

func TestClassifyRSI(t *testing.T) {
	tests := []struct {
		name string
		rsi  float64
		want signal.Status
	}{
		{"deeply oversold", 15, signal.StatusBearish},
		{"just oversold", 29.9, signal.StatusBearish},
		{"lower neutral", 30, signal.StatusNeutral},
		{"mid neutral", 50, signal.StatusNeutral},
		{"upper neutral", 70, signal.StatusNeutral},
		{"just overbought", 70.1, signal.StatusBullish},
		{"deeply overbought", 85, signal.StatusBullish},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signal.ClassifyRSI(tt.rsi); got != tt.want {
				t.Errorf("ClassifyRSI(%v) = %v, want %v", tt.rsi, got, tt.want)
			}
		})
	}
}

func TestClassifyMACD(t *testing.T) {
	tests := []struct {
		name             string
		line, signalLine float64
		want             signal.Status
	}{
		{"line above signal", 1.5, 1.0, signal.StatusBullish},
		{"line below signal", 0.5, 1.0, signal.StatusBearish},
		{"equal", 1.0, 1.0, signal.StatusNeutral},
		{"both negative line above", -0.2, -0.5, signal.StatusBullish},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := signal.ClassifyMACD(tt.line, tt.signalLine); got != tt.want {
				t.Errorf("ClassifyMACD(%v, %v) = %v, want %v", tt.line, tt.signalLine, got, tt.want)
			}
		})
	}
}

func TestClassifySMAandEMA(t *testing.T) {
	if got := signal.ClassifySMA(105, 100); got != signal.StatusBullish {
		t.Errorf("ClassifySMA above = %v, want bullish", got)
	}
	if got := signal.ClassifySMA(95, 100); got != signal.StatusBearish {
		t.Errorf("ClassifySMA below = %v, want bearish", got)
	}
	if got := signal.ClassifySMA(100, 100); got != signal.StatusNeutral {
		t.Errorf("ClassifySMA equal = %v, want neutral", got)
	}
	if got := signal.ClassifyEMA(110, 105); got != signal.StatusBullish {
		t.Errorf("ClassifyEMA above = %v, want bullish", got)
	}
	if got := signal.ClassifyEMA(100, 105); got != signal.StatusBearish {
		t.Errorf("ClassifyEMA below = %v, want bearish", got)
	}
}

func TestClassifyBollinger(t *testing.T) {
	tests := []struct {
		name                  string
		close, upper, mid, lo float64
		want                  signal.Status
	}{
		{"middle of band", 100, 110, 100, 90, signal.StatusNeutral},
		{"near upper", 109.5, 110, 100, 90, signal.StatusBearish},
		{"at upper", 110, 110, 100, 90, signal.StatusBearish},
		{"near lower", 90.5, 110, 100, 90, signal.StatusBullish},
		{"at lower", 90, 110, 100, 90, signal.StatusBullish},
		{"degenerate band", 100, 100, 100, 100, signal.StatusNeutral},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := signal.ClassifyBollinger(tt.close, tt.upper, tt.mid, tt.lo)
			if got != tt.want {
				t.Errorf("ClassifyBollinger(close=%v, u=%v, m=%v, l=%v) = %v, want %v",
					tt.close, tt.upper, tt.mid, tt.lo, got, tt.want)
			}
		})
	}
}

func TestDetailFor(t *testing.T) {
	tests := []struct {
		name      string
		indicator string
		value     float64
		status    signal.Status
		contains  string
	}{
		{"rsi oversold", signal.IndicatorRSI, 25, signal.StatusBearish, "Oversold"},
		{"rsi overbought", signal.IndicatorRSI, 75, signal.StatusBullish, "Overbought"},
		{"rsi neutral mid", signal.IndicatorRSI, 50, signal.StatusNeutral, "Neutral"},
		{"rsi approaching ob", signal.IndicatorRSI, 65, signal.StatusNeutral, "Approaching overbought"},
		{"rsi approaching os", signal.IndicatorRSI, 35, signal.StatusNeutral, "Approaching oversold"},
		{"macd bullish", signal.IndicatorMACD, 0, signal.StatusBullish, "above signal"},
		{"macd bearish", signal.IndicatorMACD, 0, signal.StatusBearish, "below signal"},
		{"sma bullish", signal.IndicatorSMA20, 105, signal.StatusBullish, "above MA"},
		{"sma bearish", signal.IndicatorSMA50, 95, signal.StatusBearish, "below MA"},
		{"ema bullish", signal.IndicatorEMA12, 110, signal.StatusBullish, "above MA"},
		{"bollinger upper", signal.IndicatorBollinger, 0, signal.StatusBearish, "upper"},
		{"bollinger lower", signal.IndicatorBollinger, 0, signal.StatusBullish, "lower"},
		{"bollinger inside", signal.IndicatorBollinger, 0, signal.StatusNeutral, "Inside"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := signal.DetailFor(tt.indicator, tt.value, tt.status)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("DetailFor(%q, %v, %v) = %q, want substring %q",
					tt.indicator, tt.value, tt.status, got, tt.contains)
			}
		})
	}
}

func TestDetailFor_Unknown(t *testing.T) {
	if got := signal.DetailFor("unknown", 1, signal.StatusBullish); got != "" {
		t.Errorf("DetailFor(unknown) = %q, want empty", got)
	}
}
