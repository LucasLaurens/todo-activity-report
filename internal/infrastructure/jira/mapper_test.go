package jira

import (
	"encoding/json"
	"testing"
)

func TestToTaskHandlesMissingFieldsAndADFAndAggregateTime(t *testing.T) {
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
	if task.Assignee != unassignedLabel {
		t.Fatalf("expected %q, got %q", unassignedLabel, task.Assignee)
	}
	if task.Priority != "" {
		t.Fatalf("expected empty priority, got %q", task.Priority)
	}
	if task.TimeSpent.Seconds() != 3600 {
		t.Fatalf("expected aggregate fallback 3600, got %d", task.TimeSpent.Seconds())
	}
	if task.ResolvedAt.IsPresent() {
		t.Fatal("expected no resolution date")
	}
	if task.RootCause != "Root via ADF" {
		t.Fatalf("expected ADF text extracted, got %q", task.RootCause)
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
	// Act
	_, err := toTask(issueDTO{Key: "HT-1", Fields: json.RawMessage(`{bad`)}, fieldMapping{})

	// Assert
	if err == nil {
		t.Fatal("expected an error on invalid fields JSON")
	}
}

func TestLooksLikeAccountID(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"712020:cde38829-9472-43d0-b093-929aeb17988d": true,
		"5b10ac8d82e05b22cc7d4ef5":                    true, // 24 hex
		"Ada Lovelace":                                false,
		"short":                                       false,
		"zzzzzzzzzzzzzzzzzzzzzzzz":                    false, // 24 non-hex
	}
	for term, want := range cases {
		if got := looksLikeAccountID(term); got != want {
			t.Fatalf("looksLikeAccountID(%q) = %v, want %v", term, got, want)
		}
	}
}

func TestAuthorSet(t *testing.T) {
	t.Parallel()
	if authorSet(nil) != nil {
		t.Fatal("expected nil for empty input")
	}
	set := authorSet([]string{"a", "b"})
	if !set["a"] || !set["b"] || len(set) != 2 {
		t.Fatalf("unexpected set: %v", set)
	}
}
