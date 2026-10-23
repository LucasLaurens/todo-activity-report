package cli_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
)

var (
	_ ports.ProgressReporter = (*cli.Spinner)(nil)
	_ ports.ProgressReporter = cli.NoopProgress{}
)

const (
	clearLine = "\r\033[K"
	// deadline is generous on purpose: it only bounds a failing test, a passing
	// one returns as soon as the condition holds.
	deadline     = 5 * time.Second
	pollInterval = 10 * time.Millisecond
)

// syncBuffer is a concurrency-safe writer for the spinner goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// waitForOutput polls instead of sleeping a fixed time, which races the
// spinner's ticker on a loaded CI machine.
func waitForOutput(t *testing.T, out *syncBuffer, want string) {
	t.Helper()
	for limit := time.Now().Add(deadline); time.Now().Before(limit); time.Sleep(pollInterval) {
		if strings.Contains(out.String(), want) {
			return
		}
	}
	t.Fatalf("expected %q in output within %s, got %q", want, deadline, out.String())
}

// returnsBeforeDeadline runs call and reports whether it returned in time, so a
// blocking call fails the test instead of hanging the suite.
func returnsBeforeDeadline(call func()) bool {
	done := make(chan struct{})
	go func() {
		call()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(deadline):
		return false
	}
}

func TestSpinnerRendersStageThenClears(t *testing.T) {
	t.Parallel()
	// Arrange
	out := &syncBuffer{}
	spinner := cli.NewSpinner(out, fr())
	spinner.Start()
	spinner.Stage("Fetching tasks")
	waitForOutput(t, out, "Fetching tasks")

	// Act
	spinner.Stop()

	// Assert
	if output := out.String(); !strings.HasSuffix(output, clearLine) {
		t.Fatalf("expected output to end with a clear sequence, got %q", output)
	}
}

func TestSpinnerStopIsIdempotent(t *testing.T) {
	t.Parallel()
	// Arrange
	out := &syncBuffer{}
	spinner := cli.NewSpinner(out, fr())
	spinner.Start()
	spinner.Stop()
	afterFirstStop := out.String()

	// Act
	returned := returnsBeforeDeadline(spinner.Stop)

	// Assert
	if !returned {
		t.Fatal("expected a second Stop to return")
	}
	if output := out.String(); output != afterFirstStop {
		t.Fatalf("expected a second Stop to write nothing, got %q after %q", output, afterFirstStop)
	}
}

func TestSpinnerStopWithoutStartReturns(t *testing.T) {
	t.Parallel()
	// Arrange
	out := &syncBuffer{}
	spinner := cli.NewSpinner(out, fr())

	// Act
	returned := returnsBeforeDeadline(spinner.Stop)

	// Assert
	if !returned {
		t.Fatal("expected Stop without Start to return")
	}
	if output := out.String(); output != clearLine {
		t.Fatalf("expected only the clear sequence, got %q", output)
	}
}
