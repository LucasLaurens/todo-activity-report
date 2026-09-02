package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFillsGapsFromSecondFile(t *testing.T) {
	// Arrange — mimic an unrelated working-dir .env shadowing the real one: the
	// first file lacks the tracker keys, the second (executable dir) provides them.
	dir := t.TempDir()
	workdirEnv := filepath.Join(dir, "workdir.env")
	binaryEnv := filepath.Join(dir, "binary.env")
	if err := os.WriteFile(workdirEnv, []byte("DOTENV_TEST_SHARED=from_workdir\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryEnv, []byte("DOTENV_TEST_SHARED=from_binary\nDOTENV_TEST_JIRA=token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_SHARED"); os.Unsetenv("DOTENV_TEST_JIRA") })

	// Act — working dir first, executable dir second (the loadDotEnv order).
	applyDotEnv(workdirEnv)
	applyDotEnv(binaryEnv)

	// Assert — first file wins its key; second fills the missing one.
	if got := os.Getenv("DOTENV_TEST_SHARED"); got != "from_workdir" {
		t.Fatalf("expected first file to win, got %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_JIRA"); got != "token" {
		t.Fatalf("expected second file to fill the missing key, got %q", got)
	}
}

func TestParseDotEnvLineSplitsOnFirstEquals(t *testing.T) {
	t.Parallel()
	// Arrange — an API token that itself contains '='.
	line := "JIRA_API_TOKEN=ATATT3xFf...kaLv7Cimok=DF9FC61E"

	// Act
	key, value, ok := parseDotEnvLine(line)

	// Assert
	if !ok || key != "JIRA_API_TOKEN" {
		t.Fatalf("expected key parsed, got ok=%v key=%q", ok, key)
	}
	if value != "ATATT3xFf...kaLv7Cimok=DF9FC61E" {
		t.Fatalf("expected value to keep trailing '=' segment, got %q", value)
	}
}

func TestParseDotEnvLineStripsSurroundingQuotes(t *testing.T) {
	t.Parallel()
	// Act
	_, value, ok := parseDotEnvLine(`JIRA_EMAIL='bot@example.com'`)

	// Assert
	if !ok || value != "bot@example.com" {
		t.Fatalf("expected unquoted value, got ok=%v value=%q", ok, value)
	}
}

func TestParseDotEnvLineSkipsBlanksAndComments(t *testing.T) {
	t.Parallel()
	// Act & Assert
	for _, line := range []string{"", "   ", "# a comment", "=novalue"} {
		if _, _, ok := parseDotEnvLine(line); ok {
			t.Fatalf("expected %q to be skipped", line)
		}
	}
}
