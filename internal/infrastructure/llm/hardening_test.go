package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClaudeAPISummarizerStripsEscapesFromAnErrorBody(t *testing.T) {
	t.Parallel()
	// Arrange — the fail-safe prints this error on stderr.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "overloaded\x1b[2J\x1b]0;owned\x07")
	}))
	defer server.Close()

	// Act
	_, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err == nil {
		t.Fatal("expected an error on HTTP 500")
	}
	if strings.ContainsRune(err.Error(), '\x1b') || strings.ContainsRune(err.Error(), '\a') {
		t.Fatalf("escape sequences reached the error: %q", err)
	}
}

func TestClaudeAPISummarizerRefusesAnOversizedResponse(t *testing.T) {
	t.Parallel()
	// Arrange — far beyond what a 1024-token answer can weigh.
	chunk := strings.Repeat(" ", 1<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for i := 0; i < 4; i++ {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	// Act
	_, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected the oversized response to be refused, got %v", err)
	}
}
