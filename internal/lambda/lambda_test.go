package lambda

import (
	"log/slog"
	"testing"
)

func TestInitLogger_ReturnsUsableLogger(t *testing.T) {
	logger := InitLogger()
	if logger == nil {
		t.Fatal("InitLogger() returned nil")
	}

	// Must not panic when used.
	logger.Info("test message", slog.String("key", "value"))
}

func TestInitLogger_LevelThresholds(t *testing.T) {
	handler := InitLogger().Handler()

	tests := []struct {
		name    string
		level   slog.Level
		enabled bool
	}{
		{name: "debug suppressed", level: slog.LevelDebug, enabled: false},
		{name: "info enabled", level: slog.LevelInfo, enabled: true},
		{name: "warn enabled", level: slog.LevelWarn, enabled: true},
		{name: "error enabled", level: slog.LevelError, enabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handler.Enabled(t.Context(), tt.level); got != tt.enabled {
				t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.enabled)
			}
		})
	}
}

func TestInitLogger_UsesJSONHandler(t *testing.T) {
	if _, ok := InitLogger().Handler().(*slog.JSONHandler); !ok {
		t.Errorf("handler type = %T, want *slog.JSONHandler", InitLogger().Handler())
	}
}

func TestInitLogger_ReturnsIndependentInstances(t *testing.T) {
	first := InitLogger()
	second := InitLogger()
	if first == second {
		t.Error("InitLogger() should return a new logger each call")
	}
}
