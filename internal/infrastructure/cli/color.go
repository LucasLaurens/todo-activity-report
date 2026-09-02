package cli

import (
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// ticketLine matches a narrative line opening on a ticket key (ACME-1042).
var ticketLine = regexp.MustCompile(`^\s*[A-Z][A-Z0-9]*-\d+`)

const (
	ansiReset = "\x1b[0m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
	ansiBlue  = "\x1b[34m"
	ansiGrey  = "\x1b[90m"
)

const noColorEnv = "NO_COLOR"

// kindColors maps what a status means (classified once, in the domain) to its
// ANSI color: what blocks is red, what is moving is blue, what shipped is
// green, what was abandoned is dimmed, and queued work keeps the terminal
// default.
var kindColors = map[domain.StatusKind]string{
	domain.StatusBlocked:    ansiRed,
	domain.StatusInFlight:   ansiBlue,
	domain.StatusDone:       ansiGreen,
	domain.StatusAbandoned:  ansiGrey,
	domain.StatusNotStarted: "",
}

func statusColor(text string) string {
	return kindColors[domain.ClassifyStatus(text)]
}

// colorEnabled keeps escape codes out of pipes, files and test buffers.
func colorEnabled(out io.Writer) bool {
	if os.Getenv(noColorEnv) != "" {
		return false
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// paintByStatus colors a status name, or a whole already-aligned line carrying
// one: escape codes must land outside tabwriter, which counts them as width.
// withTaskColor returns a renderer that colors every line it prints with the
// color of the ticket, blocking comments included.
func (r Renderer) withTaskColor(task domain.Task) Renderer {
	if r.color {
		r.lineColor = kindColors[task.StatusKind()]
	}
	return r
}

// colorizeLines colors each non-blank line separately: a color left open
// across a blank separator bleeds into the next block.
func colorizeLines(text, color string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = color + line + ansiReset
		}
	}
	return strings.Join(lines, "\n")
}

// taskColors indexes ticket colors by key. The narrative is the only ticket
// detail left in --brief, so its lines must carry the same colors as the ticket
// blocks — blocking comments included. O(n) build, O(1) lookup.
func taskColors(tasks []domain.Task) map[string]string {
	colors := make(map[string]string, len(tasks))
	for _, task := range tasks {
		colors[task.Key] = kindColors[task.StatusKind()]
	}
	return colors
}

// paintNarrativeLine colors a narrative line with the color of the ticket it
// opens on. A line naming no ticket — the overview — stays neutral, and an
// unknown key falls back to the status keywords.
func (r Renderer) paintNarrativeLine(line string, colors map[string]string) string {
	key := strings.TrimSpace(ticketLine.FindString(line))
	if key == "" {
		return line
	}
	color, known := colors[key]
	if !known {
		return r.paintByStatus(line)
	}
	if !r.color || color == "" {
		return line
	}
	return color + line + ansiReset
}

func (r Renderer) paintByStatus(text string) string {
	if !r.color {
		return text
	}
	color := statusColor(text)
	if color == "" {
		return text
	}
	return color + text + ansiReset
}
