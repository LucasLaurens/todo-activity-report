package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unsetEnvForTest removes a variable for the test only: t.Setenv records the
// original value and restores it at cleanup, the Unsetenv then makes it absent.
func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
}

func writeDotEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestApplyDotEnvSetsMissingAndKeepsExisting(t *testing.T) {
	// Arrange
	path := writeDotEnv(t, "# comment\nDOTENV_TEST_NEW=new_value\nDOTENV_TEST_EXISTING=from_file\nDOTENV_TEST_TOKEN=abc=def\n")
	t.Setenv("DOTENV_TEST_EXISTING", "from_env")
	unsetEnvForTest(t, "DOTENV_TEST_NEW")
	unsetEnvForTest(t, "DOTENV_TEST_TOKEN")

	// Act
	read := applyDotEnv(path, io.Discard)

	// Assert
	if !read {
		t.Fatal("expected file to be read")
	}
	if got := os.Getenv("DOTENV_TEST_NEW"); got != "new_value" {
		t.Fatalf("expected DOTENV_TEST_NEW set, got %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_EXISTING"); got != "from_env" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_TOKEN"); got != "abc=def" {
		t.Fatalf("expected token with '=' preserved, got %q", got)
	}
}

func TestApplyDotEnvMissingFileIsNoop(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "absent.env")

	// Act
	read := applyDotEnv(path, io.Discard)

	// Assert
	if read {
		t.Fatal("expected false for a missing file")
	}
}

func TestApplyDotEnvWarnsWhenTheReadIsCutShort(t *testing.T) {
	// Arrange — a line over the scanner's 64 KiB limit stops the read midway.
	oversized := strings.Repeat("x", 70*1024)
	path := writeDotEnv(t, "DOTENV_TEST_HUGE="+oversized+"\nDOTENV_TEST_AFTER=lost\n")
	unsetEnvForTest(t, "DOTENV_TEST_HUGE")
	unsetEnvForTest(t, "DOTENV_TEST_AFTER")
	var warnings bytes.Buffer

	// Act
	read := applyDotEnv(path, &warnings)

	// Assert
	if read {
		t.Fatal("expected a truncated read to be reported")
	}
	if !strings.Contains(warnings.String(), path) {
		t.Fatalf("expected a warning naming the file, got %q", warnings.String())
	}
}
