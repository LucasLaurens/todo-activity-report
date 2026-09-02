package cli_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
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

func TestSpinnerRendersStageThenClears(t *testing.T) {
	t.Parallel()
	// Arrange
	out := &syncBuffer{}
	spinner := cli.NewSpinner(out, fr())

	// Act
	spinner.Start()
	spinner.Stage("Fetching tasks")
	time.Sleep(250 * time.Millisecond)
	spinner.Stop()

	// Assert
	output := out.String()
	if !strings.Contains(output, "Fetching tasks") {
		t.Fatalf("expected stage label in output, got %q", output)
	}
	if !strings.HasSuffix(output, "\r\033[K") {
		t.Fatalf("expected output to end with a clear sequence, got %q", output)
	}
}

func TestSpinnerStopIsIdempotent(t *testing.T) {
	t.Parallel()
	// Arrange
	spinner := cli.NewSpinner(&syncBuffer{}, fr())
	spinner.Start()

	// Act & Assert — second Stop must not panic.
	spinner.Stop()
	spinner.Stop()
}

func TestNoopProgressStageDoesNothing(t *testing.T) {
	t.Parallel()
	// Act & Assert — just must not panic.
	cli.NoopProgress{}.Stage("anything")
}
