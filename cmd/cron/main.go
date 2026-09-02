package main

import (
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	run(logger)
}

// run executes the cron runner. Split from main so it is testable.
func run(logger *slog.Logger) {
	logger.Info("cron runner started")
	// TODO: implement cron job scheduling
	logger.Info("cron runner finished")
}
