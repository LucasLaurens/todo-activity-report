package jira

import (
	"encoding/json"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestToTaskFillsGapsOfASparseIssue(t *testing.T) {
	t.Parallel()
	// Arrange — no assignee, no priority, no timespent, no resolutiondate;
	// root cause stored as an Atlassian Document Format body.
	fields := `{
		"summary": "S",
		"status": {"name": "En cours"},
		"issuetype": {"name": "Bug"},
		"created": "2026-05-01T09:00:00.000+0000",
		"aggregatetimespent": 3600,
		"customfield_rc": {"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Root via ADF"}]}]}
	}`
	dto := issueDTO{Key: "HT-9", Fields: json.RawMessage(fields)}

	// Act
	task, err := toTask(dto, fieldMapping{rootCauseField: "customfield_rc"})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Run("missing assignee reads as unassigned", func(t *testing.T) {
		if task.Assignee != unassignedLabel {
			t.Errorf("expected %q, got %q", unassignedLabel, task.Assignee)
		}
	})
	t.Run("missing priority stays empty", func(t *testing.T) {
		if task.Priority != "" {
			t.Errorf("expected empty priority, got %q", task.Priority)
		}
	})
	t.Run("missing timespent falls back to the aggregate", func(t *testing.T) {
		if task.TimeSpent.Seconds() != 3600 {
			t.Errorf("expected aggregate fallback 3600, got %d", task.TimeSpent.Seconds())
		}
	})
	t.Run("missing resolution date leaves it unresolved", func(t *testing.T) {
		if task.ResolvedAt.IsPresent() {
			t.Error("expected no resolution date")
		}
	})
	t.Run("ADF root cause is flattened to text", func(t *testing.T) {
		if task.RootCause != "Root via ADF" {
			t.Errorf("expected ADF text extracted, got %q", task.RootCause)
		}
	})
}

func TestToTaskRejectsAnUnreadableCreationDate(t *testing.T) {
	t.Parallel()
	// Arrange — a year-1 creation date would turn the resolution delay into
	// two millennia and skew the MTTR.
	fields := `{"summary":"S","status":{"name":"Done"},"issuetype":{"name":"Bug"},"created":"yesterday"}`

	// Act
	_, err := toTask(issueDTO{Key: "HT-1", Fields: json.RawMessage(fields)}, fieldMapping{})

	// Assert
	if err == nil {
		t.Fatal("expected an error on an unreadable creation date")
	}
}

func TestToTaskPrefersTimespentOverAggregate(t *testing.T) {
	t.Parallel()
	// Arrange
	fields := `{"summary":"S","status":{"name":"Done"},"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000","timespent":1800,"aggregatetimespent":9999}`
	dto := issueDTO{Key: "HT-1", Fields: json.RawMessage(fields)}

	// Act
	task, err := toTask(dto, fieldMapping{})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.TimeSpent.Seconds() != 1800 {
		t.Fatalf("expected timespent 1800, got %d", task.TimeSpent.Seconds())
	}
}

func TestToTaskReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()
	// Act — no Arrange: the malformed payload is the whole input
	_, err := toTask(issueDTO{Key: "HT-1", Fields: json.RawMessage(`{bad`)}, fieldMapping{})

	// Assert
	if err == nil {
		t.Fatal("expected an error on invalid fields JSON")
	}
}

func TestLooksLikeAccountID(t *testing.T) {
	t.Parallel()
	// Arrange
	cases := map[string]bool{
		"712020:cde38829-9472-43d0-b093-929aeb17988d": true,
		"5b10ac8d82e05b22cc7d4ef5":                    true, // 24 hex
		"Ada Lovelace":                                false,
		"short":                                       false,
		"zzzzzzzzzzzzzzzzzzzzzzzz":                    false, // 24 non-hex
	}
	for term, want := range cases {
		t.Run(term, func(t *testing.T) {
			// Act
			got := looksLikeAccountID(term)

			// Assert
			if got != want {
				t.Errorf("looksLikeAccountID(%q) = %v, want %v", term, got, want)
			}
		})
	}
}

func TestAuthorSetIsNilWithoutAuthors(t *testing.T) {
	t.Parallel()
	// Act
	set := authorSet(nil)

	// Assert
	if set != nil {
		t.Fatalf("expected nil for empty input, got %v", set)
	}
}

func TestAuthorSetHoldsEveryAuthor(t *testing.T) {
	t.Parallel()
	// Act
	set := authorSet([]string{"a", "b"})

	// Assert
	if !set["a"] || !set["b"] || len(set) != 2 {
		t.Fatalf("unexpected set: %v", set)
	}
}

func TestSumWorklogsInPeriodRejectsAnUnreadableStart(t *testing.T) {
	t.Parallel()
	// Arrange — silently dropping the entry would under-report the day.
	period := mustPeriod(t)
	entries := []worklogEntry{{Started: "not a date", TimeSpentSeconds: 3600}}

	// Act
	_, err := sumWorklogsInPeriod(entries, period, nil)

	// Assert
	if err == nil {
		t.Fatal("expected an error on an unreadable worklog start")
	}
}

func mustPeriod(t *testing.T) domain.DateRange {
	t.Helper()
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return period
}
