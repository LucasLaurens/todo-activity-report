package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func TestRenderShowsTimePerTicketWithItsShareOfTheDay(t *testing.T) {
	t.Parallel()
	// Arrange — 3h logged over two tickets, a third one touched without time.
	report := buildReport(t,
		testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(2).Build(),
		testsupport.NewTask().WithKey("HT-2").WithTimeSpentHours(1).Build(),
		testsupport.NewTask().WithKey("HT-3").WithTimeSpentHours(0).Build(),
	)
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buffer.String()
	for _, fragment := range []string{"Répartition du temps", "HT-1", "2h 00m", "67%", "33%", "1 ticket touché sans temps loggé"} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, output)
		}
	}
	if strings.Index(output, "HT-1") > strings.Index(output, "HT-2") {
		t.Fatalf("expected the longest ticket first, got:\n%s", output)
	}
}

func TestRenderKeepsTheTimeBreakdownInBriefDetail(t *testing.T) {
	t.Parallel()
	// Arrange — brief keeps the figures, and the time breakdown is a figure.
	report := buildReport(t, testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(2).Build())
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{Detail: cli.DetailBrief}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buffer.String(), "Répartition du temps") {
		t.Fatalf("expected the time breakdown in brief, got:\n%s", buffer.String())
	}
}

func TestRenderHidesTheTimeBreakdownWhenNothingWasLogged(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t, testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(0).Build())
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buffer.String(), "Répartition du temps") {
		t.Fatalf("expected no breakdown section without logged time, got:\n%s", buffer.String())
	}
}

// The cap set by --limit is one rule for the whole report: it bounds the time
// breakdown exactly like the ticket list, and says how much it hid.
func TestRenderCapsTheTimeBreakdownWithTheTaskLimit(t *testing.T) {
	t.Parallel()
	// Arrange
	report := buildReport(t,
		testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(3).Build(),
		testsupport.NewTask().WithKey("HT-2").WithTimeSpentHours(2).Build(),
	)
	var buffer bytes.Buffer

	// Act
	err := cli.NewRenderer(&buffer, cli.RenderOptions{TaskLimit: 1}, fr()).Render(report)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	breakdown := buffer.String()[strings.Index(buffer.String(), "Répartition du temps"):]
	breakdown = breakdown[:strings.Index(breakdown, "Avancement")]
	if strings.Contains(breakdown, "HT-2") {
		t.Fatalf("expected the limit to cap the breakdown, got:\n%s", breakdown)
	}
	if !strings.Contains(breakdown, "1 de plus") {
		t.Fatalf("expected the hidden count, got:\n%s", breakdown)
	}
}
