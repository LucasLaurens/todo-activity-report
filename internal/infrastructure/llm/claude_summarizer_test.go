package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

func report(t *testing.T) domain.Report {
	t.Helper()
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return domain.Report{Project: "ACME", Period: period, Narrative: domain.None[string]()}
}

func summarizerTo(server *httptest.Server) ClaudeSummarizer {
	return ClaudeSummarizer{
		apiKey:     "k",
		model:      "m",
		endpoint:   server.URL,
		translator: i18n.NewTranslator(i18n.French),
		httpClient: server.Client(),
	}
}

func TestClaudeAPISummarizerReturnsNarrative(t *testing.T) {
	t.Parallel()
	// Arrange
	var gotKey, gotVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.Write([]byte(`{"content":[{"type":"text","text":"A concise summary."}]}`))
	}))
	defer server.Close()

	// Act
	narrative, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value, _ := narrative.Get(); value != "A concise summary." {
		t.Fatalf("unexpected narrative %q", value)
	}
	if gotKey != "k" || gotVersion != anthropicVersion {
		t.Fatalf("expected auth headers set, got key=%q version=%q", gotKey, gotVersion)
	}
}

func TestClaudeAPISummarizerErrorsOnNon200(t *testing.T) {
	t.Parallel()
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	// Act
	_, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err == nil {
		t.Fatal("expected an error on non-200")
	}
}

func TestClaudeAPISummarizerNoneOnEmptyContent(t *testing.T) {
	t.Parallel()
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"content":[]}`))
	}))
	defer server.Close()

	// Act
	narrative, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.IsPresent() {
		t.Fatal("expected absent narrative on empty content")
	}
}

func TestNewClaudeSummarizerDefaultsModel(t *testing.T) {
	t.Parallel()
	// Act
	s := NewClaudeSummarizer("key", "", i18n.NewTranslator(i18n.French), nil)

	// Assert
	if s.model != defaultModel || s.endpoint != messagesEndpoint {
		t.Fatalf("expected defaults, got model=%q endpoint=%q", s.model, s.endpoint)
	}
}
