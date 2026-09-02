package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/profitify/profitify-backend/internal/api"
	"github.com/profitify/profitify-backend/internal/config"
	"github.com/profitify/profitify-backend/internal/db"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(logger); err != nil {
		logger.Error("server exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// run starts the API server and blocks until a shutdown signal arrives.
// It returns an error rather than exiting so the startup paths are testable.
func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL, db.WithMaxConns(int32(cfg.PoolMaxConns)))
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	logger.Info("database connected")

	router := api.NewRouter(logger)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.APIPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", slog.String("port", cfg.APIPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("serving: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-done:
	}

	logger.Info("server shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutting down: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
