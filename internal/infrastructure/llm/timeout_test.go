package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// A killed CLI reports "signal: killed" with an empty stderr, which says
// nothing about the real cause: the narrative simply took longer than its
// budget.
func TestClaudeCLISummarizerNamesTheTimeoutWhenItKillsTheCLI(t *testing.T) {
	t.Parallel()
	// Arrange — a stub that outlives its budget.
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncat >/dev/null\nsleep 5\n"), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	summarizer := ClaudeCLISummarizer{binary: path, timeout: 50 * time.Millisecond, translator: i18n.NewTranslator(i18n.French)}

	// Act
	_, err := summarizer.Summarize(context.Background(), report(t))

	// Assert
	if err == nil {
		t.Fatal("expected an error when the CLI outlives its budget")
	}
	if !strings.Contains(err.Error(), "50ms") || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected the error to name the timeout, got %q", err)
	}
}
