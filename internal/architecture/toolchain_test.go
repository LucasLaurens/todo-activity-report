package architecture_test

import (
	"os"
	"regexp"
	"testing"
)

var (
	goModToolchain = regexp.MustCompile(`(?m)^toolchain go(\S+)$`)
	ciGoVersion    = regexp.MustCompile(`(?m)^\s*GO_VERSION: '([^']+)'$`)
)

// The release binaries embed the stdlib of the CI toolchain: a CI lagging behind
// go.mod would ship the security fixes the local build already has.
func TestCIBuildsWithTheGoModToolchain(t *testing.T) {
	t.Parallel()
	// Arrange
	goMod := readRepoFile(t, "../../go.mod")
	workflow := readRepoFile(t, "../../.github/workflows/ci.yml")

	// Act
	local := firstGroup(t, goModToolchain, goMod, "toolchain line in go.mod")
	ci := firstGroup(t, ciGoVersion, workflow, "GO_VERSION in ci.yml")

	// Assert
	if local != ci {
		t.Fatalf("go.mod pins toolchain go%s but CI builds with Go %s", local, ci)
	}
}

func readRepoFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func firstGroup(t *testing.T, pattern *regexp.Regexp, text, what string) string {
	t.Helper()
	match := pattern.FindStringSubmatch(text)
	if match == nil {
		t.Fatalf("no %s", what)
	}
	return match[1]
}
