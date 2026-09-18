package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// A model answer is remote content: it must never reach the terminal with
// escape sequences in it, whichever strategy produced it.
func TestClaudeAPISummarizerStripsEscapeSequencesFromTheAnswer(t *testing.T) {
	t.Parallel()
	// Arrange
	escape := string(rune(27))
	encoded, err := json.Marshal(escape + "[2JHT-1 wiped")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"content":[{"type":"text","text":%s}]}`, encoded)
	}))
	defer server.Close()

	// Act
	narrative, err := summarizerTo(server).Summarize(context.Background(), report(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	value, _ := narrative.Get()
	if value != "HT-1 wiped" {
		t.Fatalf("escape survived the API narrative: %q", value)
	}
}

func TestClaudeCLISummarizerStripsEscapeSequencesFromTheAnswer(t *testing.T) {
	t.Parallel()
	// Arrange — the stub prints an ANSI sequence, as a compromised or creative
	// model answer would.
	path := filepath.Join(t.TempDir(), "claude")
	script := "#!/bin/sh\ncat >/dev/null\nprintf 'HT-1 \\033[31mfake red\\033[0m\\n'\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	summarizer := NewClaudeCLISummarizer(path, i18n.NewTranslator(i18n.French))

	// Act
	narrative, err := summarizer.Summarize(context.Background(), report(t))

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	value, _ := narrative.Get()
	if strings.ContainsRune(value, rune(27)) {
		t.Fatalf("escape survived the CLI narrative: %q", value)
	}
	if value != "HT-1 fake red" {
		t.Fatalf("unexpected narrative: %q", value)
	}
}
