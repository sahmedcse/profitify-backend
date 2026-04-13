package domain_test

import (
	"testing"

	"github.com/profitify/profitify-backend/internal/domain"
)

func TestAllStagesLength(t *testing.T) {
	if got := len(domain.AllStages); got != 5 {
		t.Errorf("AllStages length = %d, want 5", got)
	}
}

func TestAllStagesValues(t *testing.T) {
	expected := []string{
		domain.StageIngestOHLCV,
		domain.StageFetchTechnicals,
		domain.StageFetchFundamentals,
		domain.StageEnrichTicker,
		domain.StageComputeStats,
	}
	for i, s := range domain.AllStages {
		if s != expected[i] {
			t.Errorf("AllStages[%d] = %q, want %q", i, s, expected[i])
		}
	}
}

func TestPipelineStatusConstants(t *testing.T) {
	statuses := map[string]bool{
		domain.PipelineStatusPending:   true,
		domain.PipelineStatusRunning:   true,
		domain.PipelineStatusCompleted: true,
		domain.PipelineStatusFailed:    true,
		domain.PipelineStatusSkipped:   true,
	}
	if got := len(statuses); got != 5 {
		t.Errorf("expected 5 distinct status values, got %d", got)
	}
}
