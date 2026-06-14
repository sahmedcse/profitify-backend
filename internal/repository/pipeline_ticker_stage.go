package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type pipelineTickerStageRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPipelineTickerStageRepo creates a new PipelineTickerStageRepository backed by the given connection pool.
func NewPipelineTickerStageRepo(pool *pgxpool.Pool, logger *slog.Logger) PipelineTickerStageRepository {
	return &pipelineTickerStageRepo{pool: pool, logger: logger}
}

func (r *pipelineTickerStageRepo) MarkRunning(ctx context.Context, runID, tickerID, stage string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO pipeline_ticker_stages (run_id, ticker_id, ticker, stage, status, started_at, updated_at)
		SELECT $1, $2, ticker, $3, 'running', NOW(), NOW()
		FROM pipeline_runs WHERE id = $1
		ON CONFLICT (run_id, ticker_id, stage)
		DO UPDATE SET status = 'running', started_at = NOW(), updated_at = NOW()
		RETURNING id`,
		runID, tickerID, stage).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("pipelineTickerStageRepo.MarkRunning: %w", err)
	}
	return id, nil
}

func (r *pipelineTickerStageRepo) MarkCompleted(ctx context.Context, runID, tickerID, stage string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_ticker_stages
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE run_id = $1 AND ticker_id = $2 AND stage = $3`,
		runID, tickerID, stage)
	if err != nil {
		return fmt.Errorf("pipelineTickerStageRepo.MarkCompleted: %w", err)
	}
	return nil
}

func (r *pipelineTickerStageRepo) MarkFailed(ctx context.Context, runID, tickerID, stage, errorMessage string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_ticker_stages
		SET status = 'failed', error_message = $4,
			completed_at = NOW(), updated_at = NOW()
		WHERE run_id = $1 AND ticker_id = $2 AND stage = $3`,
		runID, tickerID, stage, errorMessage)
	if err != nil {
		return fmt.Errorf("pipelineTickerStageRepo.MarkFailed: %w", err)
	}
	return nil
}

func (r *pipelineTickerStageRepo) GetByRunAndTicker(ctx context.Context, runID, tickerID string) ([]domain.PipelineTickerStage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_id, ticker_id, ticker, stage, sfn_execution_arn,
		       status, error_message, started_at, completed_at,
		       created_at, updated_at
		FROM pipeline_ticker_stages
		WHERE run_id = $1 AND ticker_id = $2
		ORDER BY stage`, runID, tickerID)
	if err != nil {
		return nil, fmt.Errorf("pipelineTickerStageRepo.GetByRunAndTicker: %w", err)
	}
	defer rows.Close()

	var out []domain.PipelineTickerStage
	for rows.Next() {
		var s domain.PipelineTickerStage
		if err := rows.Scan(
			&s.ID, &s.RunID, &s.TickerID, &s.Ticker, &s.Stage,
			&s.SFNExecutionArn, &s.Status, &s.ErrorMessage,
			&s.StartedAt, &s.CompletedAt, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("pipelineTickerStageRepo.scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pipelineTickerStageRepo.scan: %w", err)
	}
	return out, nil
}
