package llm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// The prompt budget is what makes the narrative slow: the model writes a line
// per ticket, and every comment is context it has to read first.
func TestBuildPromptStopsAtTwentyTickets(t *testing.T) {
	t.Parallel()
	// Arrange
	if maxTasksInPrompt != 20 {
		t.Fatalf("expected a 20-ticket budget, got %d", maxTasksInPrompt)
	}
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	tasks := make([]domain.Task, 23)
	for i := range tasks {
		tasks[i] = domain.Task{Key: fmt.Sprintf("HT-%d", i), Summary: "work", Status: "Done"}
	}

	// Act
	prompt := buildPrompt(domain.Report{Tasks: tasks, Period: period}, i18n.NewTranslator(i18n.French))

	// Assert
	if !strings.Contains(prompt, "and 3 more") {
		t.Fatalf("expected the overflow note, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "[HT-20]") {
		t.Fatalf("expected the 21st ticket to be cut, got:\n%s", prompt)
	}
	// Told to stay silent about the overflow, the model stops closing with
	// "the 31 remaining tickets are not detailed in the data provided".
	if !strings.Contains(prompt, "ignore them silently") {
		t.Fatalf("expected the overflow instruction, got:\n%s", prompt)
	}
}

func TestBuildPromptKeepsOnlyTheThreeMostRecentCommentsPerTicket(t *testing.T) {
	t.Parallel()
	// Arrange — five comments, oldest first, as the Jira mapper hands them over.
	comments := make([]domain.Comment, 0, 5)
	for i := 1; i <= 5; i++ {
		comments = append(comments, domain.Comment{Author: "Dev", Body: fmt.Sprintf("note %d", i)})
	}
	period, _ := domain.ParseDateRange("2026-05-01", "2026-05-31")
	task := domain.Task{Key: "HT-1", Summary: "work", Status: "Done", Comments: comments}

	// Act
	prompt := buildPrompt(domain.Report{Tasks: []domain.Task{task}, Period: period}, i18n.NewTranslator(i18n.French))

	// Assert
	for _, recent := range []string{"note 3", "note 4", "note 5"} {
		if !strings.Contains(prompt, recent) {
			t.Fatalf("expected %q in the prompt, got:\n%s", recent, prompt)
		}
	}
	for _, old := range []string{"note 1", "note 2"} {
		if strings.Contains(prompt, old) {
			t.Fatalf("expected %q to be dropped, got:\n%s", old, prompt)
		}
	}
}
