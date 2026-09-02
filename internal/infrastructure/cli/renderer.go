package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// detailTruncateLength caps short distribution labels (status / type / priority);
// they never really reach it but it guards against pathological values.
const detailTruncateLength = 200

// conciseLength bounds the default root cause / fix gist. minSentenceLength
// avoids cutting on a leading abbreviation ("e.g.") so the gist stays meaningful.
const (
	conciseLength     = 150
	minSentenceLength = 40
)

// Renderer writes a Report to a terminal-friendly, locale-aware layout. It
// depends only on io.Writer, so it is trivially testable against a buffer.
type Renderer struct {
	out   io.Writer
	opts  RenderOptions
	t     i18n.Translator
	color bool
	// lineColor colors every line printed by this renderer (set per ticket
	// block by withStatusColor); empty means the terminal default.
	lineColor string
}

// RenderOptions tunes the report layout.
//   - TaskLimit caps the detailed task list (0 = no cap).
//   - Full disables every truncation so complete root cause / fix / summaries show.
//   - Brief drops the root causes and ticket sections, keeping only the metrics
//     and the AI narrative — a numbers-plus-summary digest.
type RenderOptions struct {
	TaskLimit int
	Full      bool
	Brief     bool
}

func NewRenderer(out io.Writer, opts RenderOptions, translator i18n.Translator) Renderer {
	return Renderer{out: out, opts: opts, t: translator, color: colorEnabled(out)}
}

func (r Renderer) Render(report domain.Report) error {
	sections := []func(domain.Report) error{
		r.renderHeader,
		r.renderOverview,
		r.renderDistribution("by_status", report.Metrics.ByStatus, r.paintByStatus),
		r.renderAssignees,
		r.renderDistribution("by_type", report.Metrics.ByType, uncolored),
		r.renderDistribution("by_priority", report.Metrics.ByPriority, uncolored),
		r.renderRootCauses,
		r.renderTasks,
		r.renderNarrative,
	}
	for _, section := range sections {
		if err := section(report); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) renderHeader(report domain.Report) error {
	people := r.t.T("scope_all_users")
	if len(report.Assignees) > 0 {
		people = strings.Join(report.Assignees, ", ")
	}
	return r.printf("\n%s\n%s : %s → %s   %s : %s\n",
		r.t.T("report_title", report.Project),
		r.t.T("scope_period"), report.Period.FromString(), report.Period.ToString(),
		r.t.T("scope_users"), people)
}

func (r Renderer) renderOverview(report domain.Report) error {
	m := report.Metrics
	if err := r.section("overview"); err != nil {
		return err
	}
	return r.table(func(w io.Writer) {
		fmt.Fprintf(w, "  %s\t%d\n", r.t.T("total_tasks"), m.TotalTasks)
		fmt.Fprintf(w, "  %s\t%d (%.0f%%)\n", r.t.T("resolved"), m.ResolvedTasks, m.ResolutionRate()*100)
		fmt.Fprintf(w, "  %s\t%s\n", r.t.T("time_logged"), m.TotalTimeSpent)
		fmt.Fprintf(w, "  %s\t%s\n", r.t.T("avg_per_task"), m.AvgTimeSpent)
		fmt.Fprintf(w, "  %s\t%s\n", r.t.T("mttr"), r.formatMTTR(m.MTTR))
	})
}

func (r Renderer) renderAssignees(report domain.Report) error {
	// A single-person breakdown only repeats the overview totals.
	if len(report.Metrics.ByAssignee) <= 1 {
		return nil
	}
	if err := r.section("by_assignee"); err != nil {
		return err
	}
	return r.table(func(w io.Writer) {
		for _, entry := range report.Metrics.ByAssignee {
			fmt.Fprintf(w, "  %s\t%s\n", entry.Assignee,
				r.t.T("assignee_line", entry.Total, entry.Resolved, entry.TimeSpent))
		}
	})
}

// uncolored leaves a distribution line as-is: only statuses carry a color.
func uncolored(line string) string { return line }

func (r Renderer) renderDistribution(titleKey string, counts []domain.LabelCount, paint func(string) string) func(domain.Report) error {
	return func(_ domain.Report) error {
		if len(counts) == 0 {
			return nil
		}
		if err := r.section(titleKey); err != nil {
			return err
		}
		return r.paintedTable(paint, func(w io.Writer) {
			for _, entry := range counts {
				fmt.Fprintf(w, "%s%s\t%d\n", plainPrefix, r.clip(entry.Label, detailTruncateLength), entry.Count)
			}
		})
	}
}

// renderRootCauses lists each root cause in full, word-wrapped with a hanging
// indent. Unlike the count distributions it never truncates (the cause text is
// the point) and drops the trailing count unless a cause repeats.
func (r Renderer) renderRootCauses(report domain.Report) error {
	if r.opts.Brief {
		return nil
	}
	// Only causes hitting several tickets are worth an aggregate; unique ones are
	// already shown under their ticket. Nothing recurring → skip the section.
	var recurring []domain.LabelCount
	for _, entry := range report.Metrics.TopRootCauses {
		if entry.Count > 1 {
			recurring = append(recurring, entry)
		}
	}
	if len(recurring) == 0 {
		return nil
	}
	if err := r.section("top_root_causes"); err != nil {
		return err
	}
	for _, entry := range recurring {
		if err := r.renderBullet(fmt.Sprintf("%s (×%d)", r.concise(entry.Label), entry.Count)); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) renderTasks(report domain.Report) error {
	if r.opts.Brief {
		return nil
	}
	if len(report.Tasks) == 0 {
		return r.printf("\n\n%s\n", r.t.T("no_tasks"))
	}
	shown, hidden := r.taskWindow(report.Tasks)
	if err := r.printf("\n\n%s\n", r.t.T("tasks_header", len(shown), len(report.Tasks))); err != nil {
		return err
	}
	for _, task := range shown {
		if err := r.renderTask(task); err != nil {
			return err
		}
	}
	if hidden > 0 {
		return r.printf("\n%s\n", r.t.T("more_hidden", hidden))
	}
	return nil
}

// renderTask prints one ticket: a compact header line (key · status · time)
// followed by the summary, root cause and fix as word-wrapped indented blocks.
func (r Renderer) renderTask(task domain.Task) error {
	// The whole block takes the status color, not just the header: a blocked
	// ticket must stand out down to its cause and fix lines.
	block := r.withTaskColor(task)
	if err := block.printf("\n%s%s  ·  %s  ·  %s%s\n", bulletPrefix, task.Key, task.Status, task.TimeSpent, r.blockingMarker(task)); err != nil {
		return err
	}
	if err := block.renderIndented(task.Summary); err != nil {
		return err
	}
	if task.RootCause != "" {
		if err := block.renderIndented(fmt.Sprintf("%s: %s", r.t.T("root_cause"), r.concise(task.RootCause))); err != nil {
			return err
		}
	}
	if task.Correction != "" {
		if err := block.renderIndented(fmt.Sprintf("%s: %s", r.t.T("fix"), r.concise(task.Correction))); err != nil {
			return err
		}
	}
	return nil
}

// blockingMarker justifies the red on a ticket whose status does not say
// blocked: the blocker was reported in the comments.
func (r Renderer) blockingMarker(task domain.Task) string {
	if !task.HasBlockingComment() || domain.ClassifyStatus(task.Status) == domain.StatusBlocked {
		return ""
	}
	return "  ·  " + r.t.T("blocking_comment")
}

// renderBullet prints a word-wrapped block led by a bullet, wrapped lines
// hanging under the bullet text.
func (r Renderer) renderBullet(text string) error {
	return r.wrapLines(text, bulletPrefix, bulletContinuation)
}

// renderIndented prints a word-wrapped block indented under a ticket bullet.
func (r Renderer) renderIndented(text string) error {
	return r.wrapLines(text, bulletContinuation, bulletContinuation)
}

func (r Renderer) wrapLines(text, first, rest string) error {
	width := wrapWidth() - utf8.RuneCountInString(first)
	for i, line := range wrapText(text, width) {
		prefix := rest
		if i == 0 {
			prefix = first
		}
		if err := r.printf("%s%s\n", prefix, line); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) renderNarrative(report domain.Report) error {
	narrative, ok := report.Narrative.Get()
	if !ok {
		return nil
	}
	if err := r.printf("\n\n%s\n", r.t.T("ai_summary")); err != nil {
		return err
	}
	// One line per ticket, each colored like its own ticket block. The opening
	// overview line names every status at once, so it stays neutral.
	colors := taskColors(report.Tasks)
	for _, line := range strings.Split(narrative, "\n") {
		if err := r.printf("%s\n", r.paintNarrativeLine(line, colors)); err != nil {
			return err
		}
	}
	return nil
}

// taskWindow returns the tasks to display and how many are hidden by the limit.
func (r Renderer) taskWindow(tasks []domain.Task) ([]domain.Task, int) {
	if r.opts.TaskLimit <= 0 || len(tasks) <= r.opts.TaskLimit {
		return tasks, 0
	}
	return tasks[:r.opts.TaskLimit], len(tasks) - r.opts.TaskLimit
}

func (r Renderer) section(titleKey string) error {
	return r.printf("\n\n%s\n", r.t.T(titleKey))
}

func (r Renderer) printf(format string, args ...any) error {
	text := fmt.Sprintf(format, args...)
	if r.lineColor != "" {
		text = colorizeLines(text, r.lineColor)
	}
	_, err := fmt.Fprint(r.out, text)
	return err
}

func (r Renderer) table(write func(io.Writer)) error {
	return r.paintedTable(uncolored, write)
}

// paintedTable aligns first, colors after: tabwriter counts escape codes as
// visible width, so they must be added once the padding is computed.
func (r Renderer) paintedTable(paint func(string) string, write func(io.Writer)) error {
	var aligned strings.Builder
	w := tabwriter.NewWriter(&aligned, 0, 4, 2, ' ', 0)
	write(w)
	if err := w.Flush(); err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSuffix(aligned.String(), "\n"), "\n") {
		if err := r.printf("%s\n", paint(line)); err != nil {
			return err
		}
	}
	return nil
}

func (r Renderer) formatMTTR(mttr domain.Optional[time.Duration]) string {
	duration, ok := mttr.Get()
	if !ok {
		return r.t.T("na")
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh", hours)
}

// Row indents: bullets mark per-item lists (root causes, tickets);
// bulletContinuation aligns wrapped lines under the bullet text.
const (
	bulletPrefix       = "  • "
	bulletContinuation = "    "
	plainPrefix        = "  "
)

// defaultWrapWidth is used when the terminal width is unknown.
const defaultWrapWidth = 100

// minWrapWidth guards against absurd COLUMNS values.
const minWrapWidth = 40

// wrapWidth reads COLUMNS when available, else falls back to a sane default.
func wrapWidth() int {
	if n, err := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS"))); err == nil && n >= minWrapWidth {
		return n
	}
	return defaultWrapWidth
}

// wrapText greedily packs words into lines no wider than width runes. A single
// word longer than width overflows onto its own line rather than being split.
func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, len(words))
	current := words[0]
	currentLen := utf8.RuneCountInString(current)
	for _, word := range words[1:] {
		wordLen := utf8.RuneCountInString(word)
		if currentLen+1+wordLen > width {
			lines = append(lines, current)
			current, currentLen = word, wordLen
			continue
		}
		current += " " + word
		currentLen += 1 + wordLen
	}
	return append(lines, current)
}

// clip caps a value by runes (never mid-character on multibyte UTF-8), unless
// full output is requested — then the value is returned untouched.
func (r Renderer) clip(value string, max int) string {
	if r.opts.Full {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}

// concise reduces a long root cause / fix to its leading sentence(s) within
// conciseLength, cutting on a sentence boundary when one fits and otherwise on a
// word boundary. --full returns the complete text. Keeps the gist, drops the wall.
func (r Renderer) concise(text string) string {
	trimmed := strings.TrimSpace(text)
	if r.opts.Full {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= conciseLength {
		return trimmed
	}
	window := string(runes[:conciseLength])
	if idx := strings.LastIndex(window, ". "); idx >= minSentenceLength {
		return strings.TrimSpace(window[:idx+1])
	}
	if sp := strings.LastIndex(window, " "); sp >= minSentenceLength {
		return strings.TrimSpace(window[:sp]) + "…"
	}
	return strings.TrimSpace(string(runes[:conciseLength-1])) + "…"
}
