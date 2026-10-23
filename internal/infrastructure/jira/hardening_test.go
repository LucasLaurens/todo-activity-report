package jira_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// fetchTimeout keeps a regression into an endless pagination loop from
// hanging the suite: the test fails on the deadline instead.
const fetchTimeout = 10 * time.Second

func fetchWithDeadline(t *testing.T, handler http.HandlerFunc) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()
	_, err := newTestClient(t, handler).FetchTasks(ctx, domain.Query{Project: "ACME", Period: newPeriod(t)})
	return err
}

func TestFetchTasksStripsEscapesFromAnErrorBody(t *testing.T) {
	t.Parallel()
	// Arrange — the error body is printed on stderr by the CLI.
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "bad jql\x1b[2J\x1b]0;owned\x07")
	}

	// Act
	err := fetchWithDeadline(t, handler)

	// Assert
	if err == nil {
		t.Fatal("expected an error on HTTP 400")
	}
	if strings.ContainsRune(err.Error(), '\x1b') || strings.ContainsRune(err.Error(), '\a') {
		t.Fatalf("escape sequences reached the error: %q", err)
	}
}

func TestFetchTasksRefusesAnOversizedResponse(t *testing.T) {
	t.Parallel()
	// Arrange — a body larger than any real page streams without end.
	chunk := strings.Repeat(" ", 1<<20)
	handler := func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < 40; i++ {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
		}
	}

	// Act
	err := fetchWithDeadline(t, handler)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected the oversized response to be refused, got %v", err)
	}
}

func TestFetchTasksStopsOnARepeatedPageToken(t *testing.T) {
	t.Parallel()
	// Arrange — a server that hands back the same token forever.
	handler := routeSearchAndWorklog(`{"nextPageToken":"again","issues":[]}`, emptyWorklogPayload)

	// Act
	err := fetchWithDeadline(t, handler)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "page token") {
		t.Fatalf("expected a repeated page token to stop the fetch, got %v", err)
	}
}

func TestFetchTasksStopsAfterTooManyPages(t *testing.T) {
	t.Parallel()
	// Arrange — a fresh token on every page, never the last one.
	page := 0
	handler := func(w http.ResponseWriter, _ *http.Request) {
		page++
		fmt.Fprintf(w, `{"nextPageToken":"p%d","issues":[]}`, page)
	}

	// Act
	err := fetchWithDeadline(t, handler)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "pages") {
		t.Fatalf("expected the page cap to stop the fetch, got %v", err)
	}
}
