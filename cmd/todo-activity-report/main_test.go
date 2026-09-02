package main

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
)

func TestParseFlagsReadsValues(t *testing.T) {
	t.Parallel()
	// Act
	parsed, err := parseFlags([]string{"--date=08/06/2026", "--user=a", "--user=b,c", "--limit=5", "--no-ai"}, io.Discard)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.date != "08/06/2026" || parsed.limit != 5 || !parsed.noAI {
		t.Fatalf("unexpected flags: %+v", parsed)
	}
	if len(parsed.users) != 3 {
		t.Fatalf("expected 3 users (repeat + csv), got %v", parsed.users)
	}
}

func TestParseFlagsHelpReturnsErrHelp(t *testing.T) {
	t.Parallel()
	// Act
	_, err := parseFlags([]string{"--help"}, io.Discard)

	// Assert
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", err)
	}
}

func TestParseFlagsRejectsUnknownFlag(t *testing.T) {
	t.Parallel()
	// Act
	_, err := parseFlags([]string{"--nope"}, io.Discard)

	// Assert
	if err == nil || errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected a non-help parse error, got %v", err)
	}
}

func TestStringSliceSetTrimsAndSplits(t *testing.T) {
	t.Parallel()
	// Arrange
	var s stringSlice

	// Act
	_ = s.Set("  a , b ,, c ")

	// Assert
	if len(s) != 3 || s[0] != "a" || s[2] != "c" {
		t.Fatalf("unexpected slice: %v", s)
	}
}

func TestResolvePeriodSingleDay(t *testing.T) {
	t.Parallel()
	// Act
	period, err := resolvePeriod(flags{date: "08/06/2026"})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if period.FromString() != "2026-06-08" || period.ToString() != "2026-06-08" {
		t.Fatalf("expected single day, got %s → %s", period.FromString(), period.ToString())
	}
}

func TestResolvePeriodRange(t *testing.T) {
	t.Parallel()
	// Act
	period, err := resolvePeriod(flags{from: "01/05/2026", to: "31/05/2026"})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if period.FromString() != "2026-05-01" || period.ToString() != "2026-05-31" {
		t.Fatalf("unexpected range: %s → %s", period.FromString(), period.ToString())
	}
}

func TestResolvePeriodRejectsDateAndRangeTogether(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolvePeriod(flags{date: "08/06/2026", from: "01/05/2026"})

	// Assert
	if err == nil {
		t.Fatal("expected error when --date combined with --from/--to")
	}
}

func TestResolvePeriodRequiresBothRangeBounds(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolvePeriod(flags{from: "01/05/2026"})

	// Assert
	if err == nil {
		t.Fatal("expected error when only --from is set")
	}
}

func TestResolvePeriodRejectsBadDate(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolvePeriod(flags{date: "2026-06-08"}) // wrong format

	// Assert
	if err == nil {
		t.Fatal("expected error on non DD/MM/YYYY date")
	}
}

func TestParseDateFR(t *testing.T) {
	t.Parallel()
	// Act
	day, err := parseDateFR("08/06/2026")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if day.Year() != 2026 || day.Month() != 6 || day.Day() != 8 {
		t.Fatalf("unexpected date: %v", day)
	}
}

func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestRunHelpExitsCleanly(t *testing.T) {
	t.Parallel()
	// Act — --help prints usage and returns nil (handled ErrHelp).
	if err := run([]string{"--help"}, tempFile(t)); err != nil {
		t.Fatalf("expected nil on --help, got %v", err)
	}
}

func TestRunRejectsBadDateBeforeConfig(t *testing.T) {
	t.Parallel()
	// Act — invalid date fails at period resolution, before any config/network.
	if err := run([]string{"--date=2026-06-08"}, tempFile(t)); err == nil {
		t.Fatal("expected an error on bad date format")
	}
}

func TestRunFailsWithoutProject(t *testing.T) {
	// Arrange — valid period, but no board key anywhere.
	t.Setenv("TRACKER_PROJECT", "")

	// Act
	err := run([]string{"--date=08/06/2026"}, tempFile(t))

	// Assert
	if !errors.Is(err, errNoProject) {
		t.Fatalf("expected errNoProject, got %v", err)
	}
}

func TestRunFailsWithoutTrackerCredentials(t *testing.T) {
	// Arrange — a board is named, but the tracker adapter has no credentials.
	t.Setenv("TRACKER_PROJECT", "ACME")
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")

	// Act
	err := run([]string{"--date=08/06/2026"}, tempFile(t))

	// Assert
	if !errors.Is(err, jira.ErrMissingConfig) {
		t.Fatalf("expected jira.ErrMissingConfig, got %v", err)
	}
}

func TestResolveProjectPrefersTheFlag(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if got := resolveProject("ACME", "FALLBACK"); got != "ACME" {
		t.Fatalf("expected the flag to win, got %q", got)
	}
	if got := resolveProject("", "FALLBACK"); got != "FALLBACK" {
		t.Fatalf("expected the configured default, got %q", got)
	}
}

func TestNewProgressNoopWhenNotTerminal(t *testing.T) {
	t.Parallel()
	// Arrange
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	defer writer.Close()

	// Act — a pipe is not a TTY → no-op reporter, no-op stop.
	reporter, stop := newProgress(writer, i18n.NewTranslator(i18n.French))
	reporter.Stage("noop")
	stop()
}

func TestIsTerminalFalseForPipe(t *testing.T) {
	t.Parallel()
	// Arrange
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer reader.Close()
	defer writer.Close()

	// Act & Assert
	if isTerminal(reader) {
		t.Fatal("a pipe is not a terminal")
	}
}

func TestHelpTextDocumentsEveryFlag(t *testing.T) {
	t.Parallel()
	// Arrange
	var parsed flags
	set := newFlagSet(&parsed, io.Discard)

	// Act
	var undocumented []string
	set.VisitAll(func(f *flag.Flag) {
		if !strings.Contains(helpText, "--"+f.Name+" ") {
			undocumented = append(undocumented, f.Name)
		}
	})

	// Assert
	if len(undocumented) > 0 {
		t.Fatalf("flags missing from helpText: %v", undocumented)
	}
}

func TestRunPrintsTheVersionWithoutAnyConfiguration(t *testing.T) {
	// Arrange — no Jira credentials: asking the version must not need any.
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_API_TOKEN", "")
	out := tempFile(t)

	// Act
	err := run([]string{"--version"}, out)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	printed, readErr := os.ReadFile(out.Name())
	if readErr != nil {
		t.Fatalf("read output: %v", readErr)
	}
	expected := "todo-activity-report " + version + "\n"
	if string(printed) != expected {
		t.Fatalf("expected %q, got %q", expected, printed)
	}
}
