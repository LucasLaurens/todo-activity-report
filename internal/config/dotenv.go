package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const dotEnvFile = ".env"

// loadDotEnv loads KEY=VALUE pairs from a .env file into the process
// environment, without overriding variables already set (the real environment
// wins). It reads the working directory first, then next to the executable, and
// applies BOTH: the first file to set a key wins, so the executable-dir file
// fills any keys the working-dir file lacks. This keeps the binary working from
// any directory even when an unrelated .env sits in the working directory.
// A missing file is not an error.
func loadDotEnv() {
	for _, path := range dotEnvCandidates() {
		applyDotEnv(path)
	}
}

func dotEnvCandidates() []string {
	candidates := []string{dotEnvFile}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), dotEnvFile))
	}
	return candidates
}

// worldWritablePermission is the bit that makes a .env everyone's to rewrite.
const worldWritablePermission = 0o002

// applyDotEnv parses one file and reports whether it existed and was read.
// A world-writable .env is skipped: it holds the Jira token and the base URL,
// so anyone able to rewrite it could redirect the credentials to another host.
func applyDotEnv(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode().Perm()&worldWritablePermission != 0 {
		fmt.Fprintf(os.Stderr, "warning: ignoring world-writable %s (chmod o-w it to load it)\n", path)
		return false
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
	_ = scanner.Err()
	return true
}

// parseDotEnvLine extracts a KEY=VALUE pair, skipping blanks and comments. The
// value is split on the first '=' only (tokens may contain '=') and surrounding
// quotes are stripped.
func parseDotEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	key, value, found := strings.Cut(trimmed, "=")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	return key, unquote(strings.TrimSpace(value)), true
}

func unquote(value string) string {
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
