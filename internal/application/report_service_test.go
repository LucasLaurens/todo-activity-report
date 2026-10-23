package application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func TestGenerateProducesReportWithMetricsAndNarrative(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(3).Build(),
	}
	provider := testsupport.NewInMemoryTaskProvider(tasks...)
	summarizer := testsupport.NewInMemorySummarizer("All good this sprint.")
	service := application.NewReportService(provider, testsupport.NewPassthroughResolver(), summarizer, testsupport.NewRecordingProgress())
	query := domain.Query{Project: "ACME", Period: wholeMay(t)}

	// Act
	report, err := service.Generate(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Metrics.TotalTasks != 1 {
		t.Fatalf("expected 1 task in metrics, got %d", report.Metrics.TotalTasks)
	}
	narrative, present := report.Narrative.Get()
	if !present || narrative != "All good this sprint." {
		t.Fatalf("expected narrative to be set, got present=%v value=%q", present, narrative)
	}
}

func TestGenerateForwardsQueryToProvider(t *testing.T) {
	t.Parallel()
	// Arrange
	provider := testsupport.NewInMemoryTaskProvider()
	service := application.NewReportService(provider, testsupport.NewPassthroughResolver(), testsupport.NewSilentSummarizer(), testsupport.NewRecordingProgress())
	query := domain.Query{Project: "ACME", Assignees: []string{"Ada"}, Period: wholeMay(t)}

	// Act
	_, err := service.Generate(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.LastQuery.Project != "ACME" || len(provider.LastQuery.Assignees) != 1 {
		t.Fatalf("query not forwarded as-is, got %+v", provider.LastQuery)
	}
}

func TestGenerateResolvesAssigneeTermsToAccountIds(t *testing.T) {
	t.Parallel()
	// Arrange
	provider := testsupport.NewInMemoryTaskProvider()
	resolver := testsupport.NewInMemoryUserResolver(map[string]string{"Ada Lovelace": "712020:abc"})
	service := application.NewReportService(provider, resolver, testsupport.NewSilentSummarizer(), testsupport.NewRecordingProgress())
	query := domain.Query{Project: "ACME", Assignees: []string{"Ada Lovelace"}, Period: wholeMay(t)}

	// Act
	report, err := service.Generate(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(provider.LastQuery.Assignees) != 1 || provider.LastQuery.Assignees[0] != "712020:abc" {
		t.Fatalf("expected provider to receive resolved accountId, got %v", provider.LastQuery.Assignees)
	}
	if len(report.Assignees) != 1 || report.Assignees[0] != "Ada Lovelace" {
		t.Fatalf("expected report to keep the display term, got %v", report.Assignees)
	}
}

func TestGenerateReportsProgressStages(t *testing.T) {
	t.Parallel()
	// Arrange
	provider := testsupport.NewInMemoryTaskProvider(testsupport.NewTask().Build())
	progress := testsupport.NewRecordingProgress()
	service := application.NewReportService(provider, testsupport.NewPassthroughResolver(), testsupport.NewSilentSummarizer(), progress)

	// Act
	_, err := service.Generate(context.Background(), domain.Query{Project: "ACME", Period: wholeMay(t)})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{ports.StageResolve, ports.StageFetch, ports.StageMetrics, ports.StageSummary}
	if !slices.Equal(progress.Stages, expected) {
		t.Fatalf("expected stages %v, got %v", expected, progress.Stages)
	}
}

func TestGenerateLeavesNarrativeAbsentWhenSummarizerIsSilent(t *testing.T) {
	t.Parallel()
	// Arrange
	provider := testsupport.NewInMemoryTaskProvider(testsupport.NewTask().Build())
	service := application.NewReportService(provider, testsupport.NewPassthroughResolver(), testsupport.NewSilentSummarizer(), testsupport.NewRecordingProgress())

	// Act
	report, err := service.Generate(context.Background(), domain.Query{Project: "ACME", Period: wholeMay(t)})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Narrative.IsPresent() {
		t.Fatal("expected narrative to be absent")
	}
}

func TestGeneratePropagatesProviderError(t *testing.T) {
	t.Parallel()
	// Arrange
	wantErr := errors.New("tracker unreachable")
	provider := testsupport.NewInMemoryTaskProvider().WithError(wantErr)
	service := application.NewReportService(provider, testsupport.NewPassthroughResolver(), testsupport.NewSilentSummarizer(), testsupport.NewRecordingProgress())

	// Act
	_, err := service.Generate(context.Background(), domain.Query{Project: "ACME", Period: wholeMay(t)})

	// Assert
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
}

func TestGenerateReadsBlockingTicketsFirst(t *testing.T) {
	t.Parallel()
	// Arrange — provider order is the tracker's (updated DESC), unrelated to attention.
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-4").WithStatus("Annulé").Build(),
		testsupport.NewTask().WithKey("HT-3").WithStatus("À faire").Build(),
		testsupport.NewTask().WithKey("HT-2").WithStatus("Deployed / Completed").Build(),
		testsupport.NewTask().WithKey("HT-1").WithStatus("In progress dev").Build(),
		testsupport.NewTask().WithKey("HT-0").WithStatus("Bloqué").Build(),
	}
	service := application.NewReportService(
		testsupport.NewInMemoryTaskProvider(tasks...),
		testsupport.NewPassthroughResolver(),
		testsupport.NewSilentSummarizer(),
		testsupport.NewRecordingProgress(),
	)
	query := domain.Query{Project: "ACME", Period: wholeMay(t)}

	// Act
	report, err := service.Generate(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	keys := make([]string, 0, len(report.Tasks))
	for _, task := range report.Tasks {
		keys = append(keys, task.Key)
	}
	expected := []string{"HT-0", "HT-1", "HT-2", "HT-3", "HT-4"}
	for i, key := range expected {
		if keys[i] != key {
			t.Fatalf("expected blockers first: got %v, want %v", keys, expected)
		}
	}
}
