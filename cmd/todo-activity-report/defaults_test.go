package main

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
)

func wednesday() time.Time {
	return time.Date(2026, time.June, 10, 15, 0, 0, 0, time.Local)
}

func TestResolvePeriodWithoutAnyDateReportsOnTheLastWorkingDay(t *testing.T) {
	t.Parallel()
	// Act
	period, err := resolvePeriod(flags{}, wednesday())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if period.FromString() != "2026-06-09" || period.ToString() != "2026-06-09" {
		t.Fatalf("expected the day before, got %s → %s", period.FromString(), period.ToString())
	}
}

func TestResolvePeriodAcceptsARelativeDay(t *testing.T) {
	t.Parallel()
	// Act
	period, err := resolvePeriod(flags{date: "hier"}, wednesday())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if period.FromString() != "2026-06-09" {
		t.Fatalf("expected yesterday, got %s", period.FromString())
	}
}

func TestResolveAssigneesFallsBackToTheConfiguredAccount(t *testing.T) {
	t.Parallel()
	// Act
	assignees, err := resolveAssignees(flags{}, "lucas@example.com")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assignees) != 1 || assignees[0] != "lucas@example.com" {
		t.Fatalf("expected the configured account, got %v", assignees)
	}
}

func TestResolveAssigneesKeepsExplicitUsers(t *testing.T) {
	t.Parallel()
	// Act
	assignees, err := resolveAssignees(flags{users: stringSlice{"Ada"}}, "lucas@example.com")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assignees) != 1 || assignees[0] != "Ada" {
		t.Fatalf("expected the explicit user, got %v", assignees)
	}
}

func TestResolveAssigneesWidensToTheWholeBoardWithAll(t *testing.T) {
	t.Parallel()
	// Act
	assignees, err := resolveAssignees(flags{all: true}, "lucas@example.com")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(assignees) != 0 {
		t.Fatalf("expected no assignee filter, got %v", assignees)
	}
}

func TestResolveAssigneesRejectsAllCombinedWithUser(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolveAssignees(flags{all: true, users: stringSlice{"Ada"}}, "lucas@example.com")

	// Assert
	if !errors.Is(err, errUserWithAll) {
		t.Fatalf("expected errUserWithAll, got %v", err)
	}
}

func TestResolveRenderOptionsReadsTheDetailLevel(t *testing.T) {
	t.Parallel()
	// Act
	options, err := resolveRenderOptions(flags{detail: "brief", limit: 5}, "")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.Detail != cli.DetailBrief || options.TaskLimit != 5 {
		t.Fatalf("unexpected options: %+v", options)
	}
}

func TestResolveRenderOptionsRejectsAnUnknownDetailLevel(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolveRenderOptions(flags{detail: "verbose"}, "")

	// Assert
	if err == nil {
		t.Fatal("expected an error for an unknown --detail value")
	}
}

func TestParseFlagsReadsTouched(t *testing.T) {
	t.Parallel()
	// Act
	parsed, err := parseFlags([]string{"--touched"}, io.Discard)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parsed.touched {
		t.Fatalf("expected --touched to be read, got %+v", parsed)
	}
}

func TestResolveRenderOptionsCarriesTheTerminalWidth(t *testing.T) {
	t.Parallel()
	// Act
	options, err := resolveRenderOptions(flags{}, "132")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.Width != 132 {
		t.Fatalf("expected width 132 from COLUMNS, got %d", options.Width)
	}
}

func TestResolveRenderOptionsFallsBackOnAnUnreadableWidth(t *testing.T) {
	t.Parallel()
	// Act
	options, err := resolveRenderOptions(flags{}, "wide")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if options.Width != 0 {
		t.Fatalf("expected the default width (0), got %d", options.Width)
	}
}
