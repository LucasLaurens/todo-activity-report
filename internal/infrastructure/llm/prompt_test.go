package llm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

func TestBuildPromptIncludesMetricsAndTasks(t *testing.T) {
	t.Parallel()
	// Arrange
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	rep := domain.Report{
		Project: "ACME",
		Period:  period,
		Tasks: []domain.Task{{
			Key:        "ACME-1",
			Summary:    "Fix login",
			Assignee:   "Ada",
			RootCause:  "Expired token",
			Correction: "Rotated secret",
			TimeSpent:  domain.NewTimeSpent(7200),
		}},
		Metrics: domain.Metrics{
			TotalTasks:    1,
			ResolvedTasks: 1,
			TopRootCauses: []domain.LabelCount{{Label: "Expired token", Count: 1}},
		},
	}

	// Act
	prompt := buildPrompt(rep, i18n.NewTranslator(i18n.French))

	// Assert
	for _, want := range []string{"ACME", "2026-05-01", "2026-05-31", "ACME-1", "Expired token", "Rotated secret", "Réponds en français."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q\n%s", want, prompt)
		}
	}
}

func TestBuildPromptStatesItsLengthBudget(t *testing.T) {
	t.Parallel()
	// Arrange
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	rep := domain.Report{Project: "ACME", Period: period}

	// Act
	prompt := buildPrompt(rep, i18n.NewTranslator(i18n.French))

	// Assert
	for _, want := range []string{
		fmt.Sprintf("%d sentences of overview at most", maxOverviewSentences),
		fmt.Sprintf("%d words maximum per ticket line", maxWordsPerTicketLine),
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to state %q\n%s", want, prompt)
		}
	}
}

func TestBuildPromptCoversOpenTicketsAlongsideDone(t *testing.T) {
	t.Parallel()
	// Arrange — a done ticket with a root cause and an open one without.
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	rep := domain.Report{
		Project: "ACME",
		Period:  period,
		Tasks: []domain.Task{
			{Key: "ACME-1", Summary: "Fix login", Status: "Terminé", RootCause: "Expired token", Correction: "Rotated secret", TimeSpent: domain.NewTimeSpent(3600)},
			{Key: "ACME-2", Summary: "Build dashboard", Status: "En cours", TimeSpent: domain.NewTimeSpent(1800)},
		},
	}

	// Act
	prompt := buildPrompt(rep, i18n.NewTranslator(i18n.French))

	// Assert — every ticket is listed and the instructions cover open states.
	for _, want := range []string{"ACME-1", "ACME-2", "EVERY ticket", "KEEP THE TICKET ORDER GIVEN BELOW", "blocked", "in code review"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q\n%s", want, prompt)
		}
	}
}

func TestBuildPromptIncludesTicketComments(t *testing.T) {
	t.Parallel()
	// Arrange — a ticket carrying a comment, plus a multi-line body to collapse.
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	rep := domain.Report{
		Project: "ACME",
		Period:  period,
		Tasks: []domain.Task{{
			Key:     "ACME-9",
			Summary: "Migrate billing",
			Status:  "Bloqué",
			Comments: []domain.Comment{
				{Author: "Lucas", Body: "Waiting on\nthe vendor API key"},
			},
		}},
	}

	// Act
	prompt := buildPrompt(rep, i18n.NewTranslator(i18n.French))

	// Assert — comment reaches the model on one collapsed line, attributed.
	for _, want := range []string{"comment by Lucas", "Waiting on the vendor API key", "supporting detail"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("expected prompt to contain %q\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "Waiting on\nthe vendor") {
		t.Fatalf("expected comment body collapsed to one line\n%s", prompt)
	}
}

func TestBuildPromptTruncatesLongTaskList(t *testing.T) {
	t.Parallel()
	// Arrange — more than maxTasksInPrompt tasks.
	created := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)
	tasks := make([]domain.Task, maxTasksInPrompt+5)
	for i := range tasks {
		tasks[i] = domain.Task{Key: "HT", CreatedAt: created, TimeSpent: domain.NewTimeSpent(60)}
	}
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	rep := domain.Report{Project: "ACME", Period: period, Tasks: tasks}

	// Act
	prompt := buildPrompt(rep, i18n.NewTranslator(i18n.French))

	// Assert
	if !strings.Contains(prompt, "and 5 more") {
		t.Fatalf("expected truncation note, got:\n%s", prompt)
	}
}

// A board-wide report counts everyone's time: telling the model it is the
// assignee's time would make it narrate a false figure.
func TestBuildPromptNamesTheTimeLikeTheReportDoes(t *testing.T) {
	t.Parallel()
	// Arrange
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	boardWide := domain.Report{Project: "ACME", Period: period}
	targeted := domain.Report{Project: "ACME", Period: period, Assignees: []string{"Ada"}}
	translator := i18n.NewTranslator(i18n.French)

	// Act
	boardPrompt := buildPrompt(boardWide, translator)
	targetedPrompt := buildPrompt(targeted, translator)

	// Assert
	if strings.Contains(boardPrompt, "as assignee") {
		t.Errorf("a board-wide prompt must not claim assignee time:\n%s", boardPrompt)
	}
	if !strings.Contains(targetedPrompt, "Time logged as assignee") {
		t.Errorf("a targeted prompt must name assignee time:\n%s", targetedPrompt)
	}
}
