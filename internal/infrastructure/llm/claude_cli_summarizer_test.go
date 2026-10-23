package llm_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

func emptyReport(t *testing.T) domain.Report {
	t.Helper()
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return domain.Report{Project: "ACME", Period: period, Narrative: domain.None[string]()}
}

// writeStubBinary creates an executable shell script standing in for `claude`.
func writeStubBinary(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claude")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	return path
}

func TestClaudeCLISummarizerReturnsNarrativeFromBinary(t *testing.T) {
	t.Parallel()
	// Arrange — stub consumes stdin then prints a canned narrative.
	binary := writeStubBinary(t, "#!/bin/sh\ncat >/dev/null\necho 'Two recurring root causes.'\n")
	summarizer := llm.NewClaudeCLISummarizer(binary, i18n.NewTranslator(i18n.French))

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	value, present := narrative.Get()
	if !present || value != "Two recurring root causes." {
		t.Fatalf("expected canned narrative, got present=%v value=%q", present, value)
	}
}

func TestClaudeCLISummarizerReturnsNoneOnEmptyOutput(t *testing.T) {
	t.Parallel()
	// Arrange
	binary := writeStubBinary(t, "#!/bin/sh\ncat >/dev/null\n")
	summarizer := llm.NewClaudeCLISummarizer(binary, i18n.NewTranslator(i18n.French))

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.IsPresent() {
		t.Fatal("expected absent narrative on empty output")
	}
}

func TestClaudeCLISummarizerErrorsWhenBinaryFails(t *testing.T) {
	t.Parallel()
	// Arrange
	binary := writeStubBinary(t, "#!/bin/sh\necho 'boom' >&2\nexit 1\n")
	summarizer := llm.NewClaudeCLISummarizer(binary, i18n.NewTranslator(i18n.French))

	// Act
	_, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err == nil {
		t.Fatal("expected an error when the CLI exits non-zero")
	}
}

func TestClaudeCLISummarizerDisablesEveryToolAndProjectSetting(t *testing.T) {
	t.Parallel()
	// Arrange — the stub echoes the arguments it received.
	binary := writeStubBinary(t, "#!/bin/sh\ncat >/dev/null\necho \"$*\"\n")
	summarizer := llm.NewClaudeCLISummarizer(binary, i18n.NewTranslator(i18n.French))

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	args, _ := narrative.Get()
	want := "-p --output-format text --tools  --strict-mcp-config --setting-sources user --no-session-persistence"
	if args != want {
		t.Fatalf("expected an isolated invocation\nwant %q\ngot  %q", want, args)
	}
}

func TestClaudeCLISummarizerRunsInAnEmptyWorkingDirectory(t *testing.T) {
	t.Parallel()
	// Arrange — the stub counts the entries of its working directory.
	binary := writeStubBinary(t, "#!/bin/sh\ncat >/dev/null\nls -A | wc -l | tr -d ' '\n")
	summarizer := llm.NewClaudeCLISummarizer(binary, i18n.NewTranslator(i18n.French))

	// Act
	narrative, err := summarizer.Summarize(context.Background(), emptyReport(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries, _ := narrative.Get(); entries != "0" {
		t.Fatalf("expected an empty working directory, found %s entries", entries)
	}
}
