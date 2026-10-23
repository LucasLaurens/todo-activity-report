package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

const spinnerInterval = 100 * time.Millisecond

// clearLine returns the cursor to column 0 and erases the line.
const clearLine = "\r\033[K"

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner is an animated terminal progress indicator implementing
// ports.ProgressReporter. It renders the current stage on its own line and is
// meant for stderr, so it never pollutes the report written to stdout.
type Spinner struct {
	out       io.Writer
	t         i18n.Translator
	mu        sync.Mutex
	label     string
	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewSpinner(out io.Writer, translator i18n.Translator) *Spinner {
	return &Spinner{out: out, t: translator, stopCh: make(chan struct{}), doneCh: make(chan struct{})}
}

// Start launches the animation. Calling it again, or after Stop, does nothing.
func (s *Spinner) Start() {
	s.startOnce.Do(func() { go s.loop() })
}

// Stage receives a translation key and renders the localized label.
func (s *Spinner) Stage(key string) {
	s.mu.Lock()
	s.label = s.t.T(key)
	s.mu.Unlock()
}

// Stop halts the animation and clears the spinner line. Safe to call several
// times, and without a prior Start.
func (s *Spinner) Stop() {
	s.stopOnce.Do(func() {
		// Never started: no loop will close doneCh, so close it here and claim
		// the start, which keeps a late Start from animating a stopped spinner.
		s.startOnce.Do(func() { close(s.doneCh) })
		close(s.stopCh)
		<-s.doneCh
		fmt.Fprint(s.out, clearLine)
	})
}

func (s *Spinner) loop() {
	defer close(s.doneCh)
	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()

	for frame := 0; ; frame++ {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			label := s.label
			s.mu.Unlock()
			fmt.Fprintf(s.out, "%s%s %s", clearLine, spinnerFrames[frame%len(spinnerFrames)], label)
		}
	}
}

// NoopProgress is the ProgressReporter used when output is not a terminal.
type NoopProgress struct{}

func (NoopProgress) Stage(string) {}
