package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/profitify/profitify-backend/internal/domain"
)

type pipelineRunRepo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPipelineRunRepo creates a new PipelineRunRepository backed by the given connection pool.
func NewPipelineRunRepo(pool *pgxpool.Pool, logger *slog.Logger) PipelineRunRepository {
	return &pipelineRunRepo{pool: pool, logger: logger}
}

func (r *pipelineRunRepo) Create(ctx context.Context, run *domain.PipelineRun) (*domain.PipelineRun, error) {
	var created domain.PipelineRun
	err := r.pool.QueryRow(ctx, `
		INSERT INTO pipeline_runs (ticker_id, ticker, date, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (ticker_id, date) DO UPDATE SET updated_at = NOW()
		RETURNING id, ticker_id, ticker, date, sfn_execution_arn, status,
		          error_message, started_at, completed_at, created_at, updated_at`,
		run.TickerID, run.Ticker, run.Date, run.Status).
		Scan(&created.ID, &created.TickerID, &created.Ticker, &created.Date,
			&created.SFNExecutionArn, &created.Status, &created.ErrorMessage,
			&created.StartedAt, &created.CompletedAt,
			&created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.Create: %w", err)
	}
	return &created, nil
}

func (r *pipelineRunRepo) GetByID(ctx context.Context, id string) (*domain.PipelineRun, error) {
	var run domain.PipelineRun
	err := r.pool.QueryRow(ctx, `
		SELECT id, ticker_id, ticker, date, sfn_execution_arn, status,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM pipeline_runs
		WHERE id = $1`, id).
		Scan(&run.ID, &run.TickerID, &run.Ticker, &run.Date,
			&run.SFNExecutionArn, &run.Status, &run.ErrorMessage,
			&run.StartedAt, &run.CompletedAt,
			&run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.GetByID: %w", err)
	}
	return &run, nil
}

func (r *pipelineRunRepo) UpdateStatus(ctx context.Context, id string, status string, errorMessage string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_runs
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`,
		status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("pipelineRunRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *pipelineRunRepo) UpdateSFNArn(ctx context.Context, id, arn string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_runs
		SET sfn_execution_arn = $1, updated_at = NOW()
		WHERE id = $2`,
		arn, id)
	if err != nil {
		return fmt.Errorf("pipelineRunRepo.UpdateSFNArn: %w", err)
	}
	return nil
}

func (r *pipelineRunRepo) MarkCompleted(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_runs
		SET status = CASE
				WHEN EXISTS (
					SELECT 1 FROM pipeline_ticker_stages
					WHERE run_id = $1 AND status = 'failed'
				) THEN 'failed'
				ELSE 'completed'
			END,
			completed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("pipelineRunRepo.MarkCompleted: %w", err)
	}
	return nil
}
