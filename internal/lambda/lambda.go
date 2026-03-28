package lambda

import (
	"log/slog"
	"os"
)

// InitLogger creates a JSON slog logger suitable for AWS Lambda environments.
func InitLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
