package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
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
	// Arrange
	var out bytes.Buffer

	// Act
	_, err := parseFlags([]string{"--nope"}, &out)

	// Assert
	if err == nil || errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected a non-help parse error, got %v", err)
	}
	if printed := strings.Count(out.String(), helpText); printed != 1 {
		t.Fatalf("expected the help printed once, got %d times", printed)
	}
}

func TestStringSliceSetTrimsAndSplits(t *testing.T) {
	t.Parallel()
	// Arrange
	var s stringSlice

	// Act
	err := s.Set("  a , b ,, c ")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Fatalf("unexpected slice: %v", s)
	}
}

func TestResolvePeriodSingleDay(t *testing.T) {
	t.Parallel()
	// Act
	period, err := resolvePeriod(flags{date: "08/06/2026"}, wednesday())

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
	period, err := resolvePeriod(flags{from: "01/05/2026", to: "31/05/2026"}, wednesday())

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
	_, err := resolvePeriod(flags{date: "08/06/2026", from: "01/05/2026"}, wednesday())

	// Assert
	if !errors.Is(err, errDateWithRange) {
		t.Fatalf("expected errDateWithRange, got %v", err)
	}
}

func TestResolvePeriodRequiresBothRangeBounds(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolvePeriod(flags{from: "01/05/2026"}, wednesday())

	// Assert
	if !errors.Is(err, errIncompleteRange) {
		t.Fatalf("expected errIncompleteRange, got %v", err)
	}
}

func TestResolvePeriodRejectsBadDate(t *testing.T) {
	t.Parallel()
	// Act
	_, err := resolvePeriod(flags{date: "2026-06-08"}, wednesday()) // wrong format

	// Assert
	if err == nil {
		t.Fatal("expected error on non DD/MM/YYYY date")
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
	err := run([]string{"--help"}, tempFile(t), tempFile(t))

	// Assert
	if err != nil {
		t.Fatalf("expected nil on --help, got %v", err)
	}
}

func TestRunRejectsBadDateBeforeConfig(t *testing.T) {
	t.Parallel()
	// Act — invalid date fails at period resolution, before any config/network.
	err := run([]string{"--date=2026-06-08"}, tempFile(t), tempFile(t))

	// Assert
	if err == nil {
		t.Fatal("expected an error on bad date format")
	}
}

func TestRunFailsWithoutProject(t *testing.T) {
	// Arrange — valid period, but no board key anywhere.
	t.Setenv("TRACKER_PROJECT", "")

	// Act
	err := run([]string{"--date=08/06/2026"}, tempFile(t), tempFile(t))

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
	err := run([]string{"--date=08/06/2026"}, tempFile(t), tempFile(t))

	// Assert
	if !errors.Is(err, jira.ErrMissingConfig) {
		t.Fatalf("expected jira.ErrMissingConfig, got %v", err)
	}
}

func TestResolveProjectPrefersTheFlag(t *testing.T) {
	t.Parallel()
	// Act
	project := resolveProject("ACME", "FALLBACK")

	// Assert
	if project != "ACME" {
		t.Fatalf("expected the flag to win, got %q", project)
	}
}

func TestResolveProjectFallsBackToTheConfiguredBoard(t *testing.T) {
	t.Parallel()
	// Act
	project := resolveProject("", "FALLBACK")

	// Assert
	if project != "FALLBACK" {
		t.Fatalf("expected the configured default, got %q", project)
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

	// Act — a pipe is not a TTY → no-op reporter, no-op stop.
	reporter, stop := newProgress(writer, i18n.NewTranslator(i18n.French))
	reporter.Stage(ports.StageFetch)
	stop()

	// Assert
	if _, isNoop := reporter.(cli.NoopProgress); !isNoop {
		t.Fatalf("expected cli.NoopProgress, got %T", reporter)
	}
	writer.Close()
	drawn, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatalf("read pipe: %v", readErr)
	}
	if len(drawn) != 0 {
		t.Fatalf("expected nothing drawn on a pipe, got %q", drawn)
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
	err := run([]string{"--version"}, out, tempFile(t))

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

func TestHelpTextShowsTheDefaultTaskLimit(t *testing.T) {
	t.Parallel()
	// Arrange
	documented := fmt.Sprintf("défaut: %d ;", defaultTaskLimit)

	// Act & Assert
	if !strings.Contains(helpText, documented) {
		t.Fatalf("expected helpText to document the --limit default %q", documented)
	}
}

// inMemoryTracker composes the concrete fakes into a ports.Tracker.
type inMemoryTracker struct {
	*testsupport.InMemoryTaskProvider
	*testsupport.InMemoryUserResolver
}

func newInMemoryTracker() inMemoryTracker {
	return inMemoryTracker{testsupport.NewInMemoryTaskProvider(), testsupport.NewPassthroughResolver()}
}

func TestSelectUserResolverPassesTermsThroughWithNoResolve(t *testing.T) {
	t.Parallel()
	// Arrange
	tracker := newInMemoryTracker()

	// Act
	resolver := selectUserResolver(flags{noResolve: true}, tracker)

	// Assert
	if _, isPassthrough := resolver.(application.PassthroughResolver); !isPassthrough {
		t.Fatalf("expected a PassthroughResolver, got %T", resolver)
	}
}

func TestSelectUserResolverLetsTheTrackerResolveByDefault(t *testing.T) {
	t.Parallel()
	// Arrange
	tracker := newInMemoryTracker()

	// Act
	resolver := selectUserResolver(flags{}, tracker)

	// Assert
	if resolver != ports.UserResolver(tracker) {
		t.Fatalf("expected the tracker as resolver, got %T", resolver)
	}
}
