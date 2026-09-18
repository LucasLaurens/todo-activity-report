package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

// The total only holds worklogs on tickets assigned to the targeted people: a
// worklog on someone else's ticket never reaches the report, and the label has
// to say so rather than let it read as "everything I logged that day".
func TestRenderSaysTheLoggedTimeIsScopedToTheAssignedTickets(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t, testsupport.NewTask().WithTimeSpentHours(2).Build())
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	if !strings.Contains(output, "Temps loggé (en tant qu'assigné)") {
		t.Fatalf("expected the scoped label, got:\n%s", output)
	}
	// Short fragment: the note wraps, so a long one would straddle two lines.
	if !strings.Contains(output, "quelqu'un d'autre") {
		t.Fatalf("expected the blind-spot note, got:\n%s", output)
	}
}

func TestRenderKeepsThePlainLabelForTheWholeBoard(t *testing.T) {
	t.Parallel()
	// Arrange — no --user: every assignee of the board, so nothing is scoped out.
	report := buildReport(t, testsupport.NewTask().WithTimeSpentHours(2).Build())
	report.Assignees = nil
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	if strings.Contains(output, "en tant qu'assigné") || strings.Contains(output, "quelqu'un d'autre") {
		t.Fatalf("expected no assignee caveat for a board-wide report, got:\n%s", output)
	}
	if !strings.Contains(output, "Temps loggé") {
		t.Fatalf("expected the plain label, got:\n%s", output)
	}
}
