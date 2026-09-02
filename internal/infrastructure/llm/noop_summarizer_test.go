package llm_test

import (
	"context"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

func TestNoopSummarizerProducesNoNarrative(t *testing.T) {
	t.Parallel()
	// Act
	narrative, err := llm.NewNoopSummarizer().Summarize(context.Background(), domain.Report{})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.IsPresent() {
		t.Fatal("expected no narrative from the no-op summarizer")
	}
}
