package cli_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func fr() i18n.Translator { return i18n.NewTranslator(i18n.French) }

func buildReport(t *testing.T, tasks ...domain.Task) domain.Report {
	t.Helper()
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return domain.Report{
		Project:   "ACME",
		Period:    period,
		Assignees: []string{"Ada Lovelace"},
		Tasks:     tasks,
		Metrics:   application.NewMetricsCalculator().Calculate(tasks),
		Narrative: domain.None[string](),
	}
}

func TestRenderIncludesOverviewAndTaskDetailsInFrench(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t, testsupport.NewTask().WithKey("HT-101").WithRootCause("Expired token").Build())
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, fragment := range []string{"ACME · Rapport d'activité", "Synthèse", "Tickets traités", "• HT-101", "Cause: Expired token"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, output)
		}
	}
}

func TestRenderShowsNarrativeWhenPresent(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t, testsupport.NewTask().Build())
	report.Narrative = domain.Some("Deux causes récurrentes ce sprint.")
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buffer.String(), "Synthèse IA") {
		t.Fatalf("expected AI summary section, got:\n%s", buffer.String())
	}
}

func TestRenderBriefHidesRootCausesAndTickets(t *testing.T) {
	t.Parallel()
	// Arrange — brief keeps the metrics + narrative, drops causes and ticket list.
	report := buildReport(t, testsupport.NewTask().WithKey("HT-101").WithRootCause("Expired token").Build())
	report.Narrative = domain.Some("Synthèse parlée du jour.")
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{Brief: true}, fr())

	// Act
	if err := renderer.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	output := buffer.String()
	for _, want := range []string{"Synthèse", "Tickets traités", "Synthèse IA", "Synthèse parlée du jour."} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected %q in brief output, got:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{"Causes racines", "Tickets (", "HT-101"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("did not expect %q in brief output, got:\n%s", unwanted, output)
		}
	}
}

func TestRenderWrapsLongSummaryWithoutBreakingUTF8(t *testing.T) {
	t.Parallel()
	// Arrange — long accented summary that must wrap by words, never mid-rune.
	summary := strings.Repeat("é mot ", 40) + "MARQUEUR_FIN"
	task := testsupport.NewTask().WithKey("HT-1").Build()
	task.Summary = summary
	report := buildReport(t, task)
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	if !utf8.ValidString(output) {
		t.Fatal("output contains invalid UTF-8 — a rune was cut mid-character")
	}
	if strings.Contains(output, "…") {
		t.Fatalf("summaries wrap, they are not truncated, got:\n%s", output)
	}
	if !strings.Contains(output, "MARQUEUR_FIN") {
		t.Fatal("expected the full summary (its end marker) to be shown")
	}
}

func TestRenderFullSkipsTruncation(t *testing.T) {
	t.Parallel()
	// Arrange — a long root cause that would be clipped in default mode.
	rootCause := strings.Repeat("cause détaillée ", 40) + "MARQUEUR_FIN"
	report := buildReport(t, testsupport.NewTask().WithKey("HT-1").WithRootCause(rootCause).Build())
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{Full: true}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	if !strings.Contains(output, "MARQUEUR_FIN") {
		t.Fatal("expected the complete root cause (its end marker) in full mode")
	}
	if strings.Contains(output, "…") {
		t.Fatalf("expected no ellipsis in full mode, got:\n%s", output)
	}
}

func rootCauseSection(output string) string {
	if start := strings.Index(output, "Causes racines"); start >= 0 {
		output = output[start:]
	}
	if end := strings.Index(output, "Tickets ("); end >= 0 {
		output = output[:end]
	}
	return output
}

func TestRenderRootCausesConciseByDefault(t *testing.T) {
	t.Parallel()
	// Arrange — a recurring cause (two tickets) whose gist is its first sentence.
	rootCause := "Le compteur lit deux endpoints divergents. " + strings.Repeat("détail technique supplémentaire ", 30)
	report := buildReport(t,
		testsupport.NewTask().WithKey("HT-1").WithRootCause(rootCause).Build(),
		testsupport.NewTask().WithKey("HT-2").WithRootCause(rootCause).Build(),
	)
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	section := rootCauseSection(buffer.String())
	if !strings.Contains(section, "Le compteur lit deux endpoints divergents. (×2)") {
		t.Fatalf("expected the gist with recurrence count, got:\n%s", section)
	}
	if strings.Contains(section, "détail technique supplémentaire") {
		t.Fatal("expected the long tail to be dropped in concise mode")
	}
}

func TestRenderRootCausesShowsFullTextWrapped(t *testing.T) {
	// Arrange — full mode shows the complete recurring cause, wrapped to a fixed width.
	t.Setenv("COLUMNS", "60")
	rootCause := strings.Repeat("analyse technique détaillée ", 12) + "MARQUEUR_FINAL"
	report := buildReport(t,
		testsupport.NewTask().WithKey("HT-1").WithRootCause(rootCause).Build(),
		testsupport.NewTask().WithKey("HT-2").WithRootCause(rootCause).Build(),
	)
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{Full: true}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	section := rootCauseSection(buffer.String())
	if strings.Contains(section, "…") {
		t.Fatalf("full mode must not truncate, got:\n%s", section)
	}
	if !strings.Contains(section, "MARQUEUR_FINAL") {
		t.Fatal("expected the end of the root cause to be visible (full text)")
	}
	if !strings.Contains(section, "  • analyse") {
		t.Fatal("expected a bullet on the first root-cause line")
	}
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "  • analyse") || strings.HasPrefix(line, "    analyse") {
			if width := utf8.RuneCountInString(line); width > 60 {
				t.Fatalf("wrapped line exceeds width 60 (%d): %q", width, line)
			}
		}
	}
}

func TestRenderCapsTaskListAndReportsHidden(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").Build(),
		testsupport.NewTask().WithKey("HT-2").Build(),
		testsupport.NewTask().WithKey("HT-3").Build(),
	}
	report := buildReport(t, tasks...)
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{TaskLimit: 1}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	if !strings.Contains(output, "Tickets (1 sur 3)") {
		t.Fatalf("expected windowed header, got:\n%s", output)
	}
	if !strings.Contains(output, "2 de plus") {
		t.Fatalf("expected hidden-count note, got:\n%s", output)
	}
	if strings.Contains(output, "HT-3") {
		t.Fatalf("expected HT-3 to be hidden by the limit, got:\n%s", output)
	}
}

func TestRenderHandlesEmptyTaskSet(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t)
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr())

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buffer.String(), "Aucun ticket traité") {
		t.Fatalf("expected empty-state message, got:\n%s", buffer.String())
	}
}

func TestRenderEnglishLocale(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t, testsupport.NewTask().WithKey("HT-1").Build())
	var buffer bytes.Buffer
	renderer := cli.NewRenderer(&buffer, cli.RenderOptions{}, i18n.NewTranslator(i18n.English))

	// Act
	err := renderer.Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buffer.String(), "Activity report") || !strings.Contains(buffer.String(), "Overview") {
		t.Fatalf("expected English labels, got:\n%s", buffer.String())
	}
}
