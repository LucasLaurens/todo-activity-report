package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDotEnvRefusesAWorldWritableFile(t *testing.T) {
	// Arrange — a .env anyone can rewrite could redirect JIRA_BASE_URL to an
	// attacker's host and harvest the API token.
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DOTENV_WORLD_WRITABLE=leaked\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(path, 0o666); err != nil { // explicit: WriteFile is umask-filtered
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Unsetenv("DOTENV_WORLD_WRITABLE") })

	// Act
	read := applyDotEnv(path)

	// Assert
	if read {
		t.Fatal("expected a world-writable .env to be skipped")
	}
	if value := os.Getenv("DOTENV_WORLD_WRITABLE"); value != "" {
		t.Fatalf("expected no variable to be set, got %q", value)
	}
}

func TestApplyDotEnvReadsAPrivateFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DOTENV_PRIVATE=kept\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Cleanup(func() { os.Unsetenv("DOTENV_PRIVATE") })

	// Act
	read := applyDotEnv(path)

	// Assert
	if !read {
		t.Fatal("expected a private .env to be read")
	}
	if value := os.Getenv("DOTENV_PRIVATE"); value != "kept" {
		t.Fatalf("expected the variable to be set, got %q", value)
	}
}
