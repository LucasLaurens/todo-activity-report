package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDotEnvSetsMissingAndKeepsExisting(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "# comment\nNEW_KEY=new_value\nEXISTING_KEY=from_file\nTOKEN=abc=def\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("EXISTING_KEY", "from_env")
	os.Unsetenv("NEW_KEY")
	os.Unsetenv("TOKEN")

	// Act
	ok := applyDotEnv(path)

	// Assert
	if !ok {
		t.Fatal("expected file to be read")
	}
	if got := os.Getenv("NEW_KEY"); got != "new_value" {
		t.Fatalf("expected NEW_KEY set, got %q", got)
	}
	if got := os.Getenv("EXISTING_KEY"); got != "from_env" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
	if got := os.Getenv("TOKEN"); got != "abc=def" {
		t.Fatalf("expected token with '=' preserved, got %q", got)
	}
}

func TestApplyDotEnvMissingFileIsNoop(t *testing.T) {
	// Act & Assert
	if applyDotEnv(filepath.Join(t.TempDir(), "absent.env")) {
		t.Fatal("expected false for a missing file")
	}
}
