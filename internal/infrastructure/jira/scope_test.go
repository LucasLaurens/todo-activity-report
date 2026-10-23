package jira_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func captureJQL(t *testing.T, query domain.Query) string {
	t.Helper()
	var captured string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("jql")
		w.Write([]byte(`{"issues":[]}`))
	})
	if _, err := client.FetchTasks(context.Background(), query); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return captured
}

// By default the report is about work done: time logged or a status moved. A
// bare `updated` match also catches a bulk field edit or someone else's
// comment, which drowned a one-day report in tickets nobody worked on.
func TestFetchTasksScopesToRealWorkByDefault(t *testing.T) {
	t.Parallel()
	// Act
	jql := captureJQL(t, domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	for _, expected := range []string{`worklogDate >= "2026-05-01"`, `status CHANGED DURING`} {
		if !strings.Contains(jql, expected) {
			t.Fatalf("expected JQL to contain %q, got %q", expected, jql)
		}
	}
	if strings.Contains(jql, "updated >=") {
		t.Fatalf("expected no bare updated clause by default, got %q", jql)
	}
}

func TestFetchTasksWidensToEveryUpdateWhenAsked(t *testing.T) {
	t.Parallel()
	// Act
	jql := captureJQL(t, domain.Query{Project: "ACME", Period: newPeriod(t), IncludeTouched: true})

	// Assert
	if !strings.Contains(jql, `updated >= "2026-05-01 00:00" AND updated <= "2026-05-31 23:59"`) {
		t.Fatalf("expected the updated clause, got %q", jql)
	}
	if !strings.Contains(jql, "worklogDate >=") {
		t.Fatalf("expected the worklog clause to remain, got %q", jql)
	}
}
