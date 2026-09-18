package llm_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

// The narrative is a bonus over metrics that are already computed: losing it
// must not lose the whole report — a timed-out `claude` CLI used to make the
// command exit 1 with nothing printed.
func TestFailSafeSummarizerKeepsTheReportWhenTheNarrativeFails(t *testing.T) {
	t.Parallel()
	// Arrange
	var warnings bytes.Buffer
	broken := testsupport.NewInMemorySummarizer("unused").WithError(errors.New("claude cli timed out after 4m0s"))
	summarizer := llm.NewFailSafeSummarizer(broken, &warnings)

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("expected the failure to be swallowed, got %v", err)
	}
	if narrative.IsPresent() {
		t.Fatalf("expected no narrative, got %q", narrative.OrElse(""))
	}
	if !strings.Contains(warnings.String(), "timed out") {
		t.Fatalf("expected the cause on the warning channel, got %q", warnings.String())
	}
}

func TestFailSafeSummarizerPassesASuccessfulNarrativeThrough(t *testing.T) {
	t.Parallel()
	// Arrange
	var warnings bytes.Buffer
	summarizer := llm.NewFailSafeSummarizer(testsupport.NewInMemorySummarizer("Deux causes récurrentes."), &warnings)

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.OrElse("") != "Deux causes récurrentes." {
		t.Fatalf("expected the narrative to pass through, got %q", narrative.OrElse(""))
	}
	if warnings.Len() != 0 {
		t.Fatalf("expected no warning on success, got %q", warnings.String())
	}
}
