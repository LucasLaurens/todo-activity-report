package jira_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
)

func newPeriod(t *testing.T) domain.DateRange {
	t.Helper()
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return period
}

const singlePagePayload = `{
  "startAt": 0, "maxResults": 100, "total": 1,
  "issues": [{
    "key": "HT-101",
    "fields": {
      "summary": "Fix login",
      "status": {"name": "Done"},
      "assignee": {"displayName": "Ada Lovelace"},
      "issuetype": {"name": "Bug"},
      "priority": {"name": "High"},
      "created": "2026-05-01T09:00:00.000+0000",
      "resolutiondate": "2026-05-01T13:00:00.000+0000",
      "timespent": 7200,
      "customfield_100": "Expired token",
      "customfield_200": "Refreshed the secret"
    }
  }]
}`

const worklogPayload = `{
  "startAt": 0, "total": 1,
  "worklogs": [
    {"started": "2026-05-10T10:00:00.000+0000", "timeSpentSeconds": 7200, "author": {"accountId": "u1"}}
  ]
}`

func TestFetchTasksMapsIssueIncludingCustomFields(t *testing.T) {
	t.Parallel()
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/worklog") {
			w.Write([]byte(worklogPayload))
			return
		}
		w.Write([]byte(singlePagePayload))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{
		BaseURL:         server.URL,
		Email:           "bot@example.com",
		APIToken:        "token",
		RootCauseField:  "customfield_100",
		CorrectionField: "customfield_200",
	}, server.Client())

	// Act
	tasks, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	task := tasks[0]
	if task.Key != "HT-101" || task.Assignee != "Ada Lovelace" {
		t.Fatalf("unexpected mapping: %+v", task)
	}
	if task.RootCause != "Expired token" || task.Correction != "Refreshed the secret" {
		t.Fatalf("custom fields not mapped: rc=%q fix=%q", task.RootCause, task.Correction)
	}
	if task.TimeSpent.Hours() != 2 {
		t.Fatalf("expected 2h time spent, got %v", task.TimeSpent.Hours())
	}
	if !task.ResolvedAt.IsPresent() {
		t.Fatal("expected task to carry its resolution date")
	}
}

func TestFetchTasksMapsRecentCommentsFromADF(t *testing.T) {
	t.Parallel()
	// Arrange — six comments, oldest first; bodies in Atlassian Document Format.
	commentEntries := make([]string, 0, 6)
	for i := 1; i <= 6; i++ {
		body := fmt.Sprintf(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"note %d"}]}]}`, i)
		commentEntries = append(commentEntries, fmt.Sprintf(`{"author":{"displayName":"Dev %d"},"body":%s}`, i, body))
	}
	search := fmt.Sprintf(`{"issues":[{"key":"HT-7","fields":{"summary":"a","status":{"name":"En cours"},"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000","comment":{"comments":[%s]}}}]}`,
		strings.Join(commentEntries, ","))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/worklog") {
			w.Write([]byte(`{"startAt":0,"total":0,"worklogs":[]}`))
			return
		}
		w.Write([]byte(search))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	tasks, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert — only the 5 most recent survive, ADF flattened, author kept.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	comments := tasks[0].Comments
	if len(comments) != 5 {
		t.Fatalf("expected 5 recent comments, got %d", len(comments))
	}
	if comments[0].Body != "note 2" || comments[4].Body != "note 6" {
		t.Fatalf("expected tail of comments flattened, got %+v", comments)
	}
	if comments[4].Author != "Dev 6" {
		t.Fatalf("expected comment author kept, got %q", comments[4].Author)
	}
}

func TestFetchTasksRequestsCommentField(t *testing.T) {
	t.Parallel()
	// Arrange
	var capturedFields string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedFields = r.URL.Query().Get("fields")
		w.Write([]byte(`{"issues":[]}`))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	_, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert — comments come inline with the search, sparing a per-ticket call.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedFields, "comment") {
		t.Fatalf("expected comment field requested inline, got %q", capturedFields)
	}
}

func TestFetchTasksCountsOnlyInPeriodWorklogOfTargetedAuthor(t *testing.T) {
	t.Parallel()
	// Arrange — one issue; worklog mixes in/out of period and two authors.
	search := `{"issues":[{"key":"HT-1","fields":{"summary":"a","status":{"name":"En cours"},"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000"}}]}`
	worklog := `{"startAt":0,"total":3,"worklogs":[
		{"started":"2026-05-10T10:00:00.000+0000","timeSpentSeconds":3600,"author":{"accountId":"u1"}},
		{"started":"2026-05-10T11:00:00.000+0000","timeSpentSeconds":1800,"author":{"accountId":"u2"}},
		{"started":"2026-04-30T10:00:00.000+0000","timeSpentSeconds":7200,"author":{"accountId":"u1"}}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/worklog") {
			w.Write([]byte(worklog))
			return
		}
		w.Write([]byte(search))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())
	query := domain.Query{Project: "ACME", Assignees: []string{"u1"}, Period: newPeriod(t)}

	// Act
	tasks, err := client.FetchTasks(context.Background(), query)

	// Assert — only u1's in-period worklog (3600s) counts; u2 and April excluded.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TimeSpent.Seconds() != 3600 {
		t.Fatalf("expected 3600s logged, got %+v", tasks)
	}
}

func TestFetchTasksSendsScopedJQLWithBasicAuth(t *testing.T) {
	t.Parallel()
	// Arrange
	var capturedJQL string
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedJQL = r.URL.Query().Get("jql")
		capturedAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"startAt":0,"maxResults":100,"total":0,"issues":[]}`))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())
	query := domain.Query{Project: "ACME", Assignees: []string{"Ada"}, Period: newPeriod(t)}

	// Act
	_, err := client.FetchTasks(context.Background(), query)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, fragment := range []string{
		`project = "ACME"`,
		`assignee in ("Ada")`,
		`worklogDate >= "2026-05-01"`,
		`worklogDate <= "2026-05-31"`,
		`status CHANGED DURING ("2026-05-01 00:00", "2026-05-31 23:59")`,
	} {
		if !strings.Contains(capturedJQL, fragment) {
			t.Fatalf("expected JQL to contain %q, got %q", fragment, capturedJQL)
		}
	}
	if !strings.HasPrefix(capturedAuth, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", capturedAuth)
	}
}

func TestFetchTasksPaginatesUntilTokenExhausted(t *testing.T) {
	t.Parallel()
	// Arrange
	firstPage := `{"nextPageToken":"page2","issues":[{"key":"HT-1","fields":{"summary":"a","status":{"name":"Done"},"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000"}}]}`
	secondPage := `{"issues":[{"key":"HT-2","fields":{"summary":"b","status":{"name":"Done"},"issuetype":{"name":"Bug"},"created":"2026-05-02T09:00:00.000+0000"}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("nextPageToken") == "" {
			w.Write([]byte(firstPage))
			return
		}
		w.Write([]byte(secondPage))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	tasks, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks across pages, got %d", len(tasks))
	}
}

func TestResolveTurnsNameIntoAccountID(t *testing.T) {
	t.Parallel()
	// Arrange
	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("query")
		w.Write([]byte(`[{"accountId":"712020:cde","displayName":"Ada Lovelace"}]`))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	accountID, err := client.Resolve(context.Background(), "Ada Lovelace")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accountID != "712020:cde" {
		t.Fatalf("expected resolved accountId, got %q", accountID)
	}
	if capturedQuery != "Ada Lovelace" {
		t.Fatalf("expected search query forwarded, got %q", capturedQuery)
	}
}

func TestResolveReturnsAccountIDTermUnchanged(t *testing.T) {
	t.Parallel()
	// Arrange — server would fail the test if hit, since accountIds skip search.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("user search must not be called for an accountId term")
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	accountID, err := client.Resolve(context.Background(), "712020:cde38829-9472-43d0-b093-929aeb17988d")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accountID != "712020:cde38829-9472-43d0-b093-929aeb17988d" {
		t.Fatalf("expected term unchanged, got %q", accountID)
	}
}

func TestResolveErrorsWhenNoUserMatches(t *testing.T) {
	t.Parallel()
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	_, err := client.Resolve(context.Background(), "Nobody")

	// Assert
	if err == nil {
		t.Fatal("expected an error when no user matches")
	}
}

func TestFetchTasksPropagatesWorklogError(t *testing.T) {
	t.Parallel()
	// Arrange — search succeeds, worklog fails.
	search := `{"issues":[{"key":"HT-1","fields":{"summary":"a","status":{"name":"Done"},"issuetype":{"name":"Bug"},"created":"2026-05-01T09:00:00.000+0000"}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/worklog") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(search))
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	_, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	if err == nil {
		t.Fatal("expected worklog error to propagate")
	}
}

func TestFetchTasksReturnsErrorOnNon200(t *testing.T) {
	t.Parallel()
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := jira.NewClient(jira.Config{BaseURL: server.URL, Email: "bot@example.com", APIToken: "token"}, server.Client())

	// Act
	_, err := client.FetchTasks(context.Background(), domain.Query{Project: "ACME", Period: newPeriod(t)})

	// Assert
	if err == nil {
		t.Fatal("expected an error on HTTP 500")
	}
}
