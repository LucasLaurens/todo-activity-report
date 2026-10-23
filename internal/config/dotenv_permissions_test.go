package config

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestApplyDotEnvRefusesAWorldWritableFile(t *testing.T) {
	// Arrange — a .env anyone can rewrite could redirect JIRA_BASE_URL to an
	// attacker's host and harvest the API token.
	path := writeDotEnv(t, "DOTENV_TEST_WORLD_WRITABLE=leaked\n")
	if err := os.Chmod(path, 0o666); err != nil { // explicit: WriteFile is umask-filtered
		t.Fatalf("chmod: %v", err)
	}
	unsetEnvForTest(t, "DOTENV_TEST_WORLD_WRITABLE")
	var warnings bytes.Buffer

	// Act
	read := applyDotEnv(path, &warnings)

	// Assert
	if read {
		t.Fatal("expected a world-writable .env to be skipped")
	}
	if value := os.Getenv("DOTENV_TEST_WORLD_WRITABLE"); value != "" {
		t.Fatalf("expected no variable to be set, got %q", value)
	}
	if !strings.Contains(warnings.String(), "world-writable") {
		t.Fatalf("expected a world-writable warning, got %q", warnings.String())
	}
}

func TestApplyDotEnvReadsAPrivateFile(t *testing.T) {
	// Arrange
	path := writeDotEnv(t, "DOTENV_TEST_PRIVATE=kept\n")
	unsetEnvForTest(t, "DOTENV_TEST_PRIVATE")

	// Act
	read := applyDotEnv(path, io.Discard)

	// Assert
	if !read {
		t.Fatal("expected a private .env to be read")
	}
	if value := os.Getenv("DOTENV_TEST_PRIVATE"); value != "kept" {
		t.Fatalf("expected the variable to be set, got %q", value)
	}
}
