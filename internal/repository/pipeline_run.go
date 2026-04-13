package repository

import (
	"context"
	"encoding/json"
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
	paramsJSON, err := json.Marshal(run.RunParams)
	if err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.Create: marshaling run_params: %w", err)
	}

	var created domain.PipelineRun
	var paramsBytes []byte
	err = r.pool.QueryRow(ctx, `
		INSERT INTO pipeline_runs (run_params, sfn_execution_arn, status, ticker_count)
		VALUES ($1, $2, $3, $4)
		RETURNING id, run_params, sfn_execution_arn, status, ticker_count,
		          completed_count, failed_count, error_message,
		          started_at, completed_at, created_at, updated_at`,
		paramsJSON, run.SFNExecutionArn, run.Status, run.TickerCount).
		Scan(&created.ID, &paramsBytes, &created.SFNExecutionArn,
			&created.Status, &created.TickerCount, &created.CompletedCount,
			&created.FailedCount, &created.ErrorMessage,
			&created.StartedAt, &created.CompletedAt,
			&created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.Create: %w", err)
	}

	if err := json.Unmarshal(paramsBytes, &created.RunParams); err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.Create: unmarshaling run_params: %w", err)
	}
	return &created, nil
}

func (r *pipelineRunRepo) GetByID(ctx context.Context, id string) (*domain.PipelineRun, error) {
	var run domain.PipelineRun
	var paramsBytes []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, run_params, sfn_execution_arn, status, ticker_count,
		       completed_count, failed_count, error_message,
		       started_at, completed_at, created_at, updated_at
		FROM pipeline_runs
		WHERE id = $1`, id).
		Scan(&run.ID, &paramsBytes, &run.SFNExecutionArn,
			&run.Status, &run.TickerCount, &run.CompletedCount,
			&run.FailedCount, &run.ErrorMessage,
			&run.StartedAt, &run.CompletedAt,
			&run.CreatedAt, &run.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.GetByID: %w", err)
	}

	if err := json.Unmarshal(paramsBytes, &run.RunParams); err != nil {
		return nil, fmt.Errorf("pipelineRunRepo.GetByID: unmarshaling run_params: %w", err)
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

func (r *pipelineRunRepo) UpdateCounts(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_runs
		SET completed_count = (
				SELECT COUNT(DISTINCT ticker_id) FROM pipeline_ticker_stages
				WHERE run_id = $1
				AND ticker_id NOT IN (
					SELECT ticker_id FROM pipeline_ticker_stages
					WHERE run_id = $1 AND status <> 'completed'
				)
			),
			failed_count = (
				SELECT COUNT(DISTINCT ticker_id) FROM pipeline_ticker_stages
				WHERE run_id = $1 AND status = 'failed'
			),
			updated_at = NOW()
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("pipelineRunRepo.UpdateCounts: %w", err)
	}
	return nil
}

func (r *pipelineRunRepo) MarkCompleted(ctx context.Context, id string) error {
	if err := r.UpdateCounts(ctx, id); err != nil {
		return fmt.Errorf("pipelineRunRepo.MarkCompleted: %w", err)
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE pipeline_runs
		SET status = CASE
				WHEN failed_count > 0 THEN 'failed'
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
