package jira_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
)

// jsonString encodes a Go string as a JSON string literal, so a test payload can
// carry control characters without putting raw ones in the source file.
func jsonString(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return string(encoded)
}

func TestFetchTasksStripsTerminalEscapesComingFromTheTracker(t *testing.T) {
	t.Parallel()
	// Arrange — a summary, a status, a custom field and a comment carrying escape
	// sequences: printed raw, they would repaint or erase the report.
	escape := string(rune(27))
	bell := string(rune(7))
	search := fmt.Sprintf(`{"issues":[{"key":"HT-66","fields":{`+
		`"summary":%s,"status":{"name":%s},`+
		`"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000",`+
		`"customfield_100":%s,`+
		`"comment":{"comments":[{"author":{"displayName":"Dev"},"body":%s}]}}}]}`,
		jsonString(t, escape+"[2Jwiped"),
		jsonString(t, escape+"[31mEn cours"),
		jsonString(t, bell+"Config drift"),
		jsonString(t, escape+"[2Kspoof"),
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/worklog") {
			w.Write([]byte(`{"startAt":0,"total":0,"worklogs":[]}`))
			return
		}
		w.Write([]byte(search))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{
		BaseURL:        server.URL,
		Email:          "bot@example.com",
		APIToken:       "token",
		RootCauseField: "customfield_100",
	}, server.Client())

	// Act
	tasks, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	task := tasks[0]
	if task.Summary != "wiped" || task.Status != "En cours" || task.RootCause != "Config drift" {
		t.Fatalf("escapes survived mapping: %+v", task)
	}
	if len(task.Comments) != 1 || task.Comments[0].Body != "spoof" {
		t.Fatalf("escapes survived in comments: %+v", task.Comments)
	}
}
