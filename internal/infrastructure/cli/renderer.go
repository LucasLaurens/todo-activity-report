package cli

import (
	"fmt"
	"io"
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

// breakdownSummaryLength keeps a time-breakdown row on one terminal line: the
// key, the time and the share already take a third of the width.
const breakdownSummaryLength = 60

const (
	percentPerUnit = 100
	hoursPerDay    = 24
)

// Table layout handed to tabwriter: columns sized to their content, two spaces
// between them.
const (
	tableMinCellWidth = 0
	tableTabWidth     = 4
	tablePadding      = 2
	tablePadChar      = ' '
	tableFlags        = 0
)

// fieldSeparator joins the fields of a ticket header line.
const fieldSeparator = "  ·  "

// Renderer writes a Report to a terminal-friendly, locale-aware layout. It
// depends only on io.Writer, so it is trivially testable against a buffer.
type Renderer struct {
	out   io.Writer
	opts  RenderOptions
	t     i18n.Translator
	color bool
	// page accumulates the report so that writing to out, the only step that
	// can fail, happens once at the end of Render.
	page *strings.Builder
	// lineColor colors every line printed by this renderer (set per ticket
	// block by withTaskColor); empty means the terminal default.
	lineColor string
}

// RenderOptions tunes the report layout.
//   - TaskLimit caps the detailed task list and the time breakdown (0 = no cap).
//   - Detail picks how much is printed: brief figures, gists, or full texts.
//   - Width is the terminal width wrapped text must fit in; 0, or anything
//     below minWrapWidth, falls back to defaultWrapWidth.
type RenderOptions struct {
	TaskLimit int
	Detail    DetailLevel
	Width     int
}

func NewRenderer(out io.Writer, opts RenderOptions, translator i18n.Translator) Renderer {
	return Renderer{out: out, opts: opts, t: translator, color: colorEnabled(out)}
}

func (r Renderer) Render(report domain.Report) error {
	r.page = &strings.Builder{}
	sections := []func(domain.Report){
		r.renderHeader,
		r.renderOverview,
		r.renderTimeScopeNote,
		r.renderTimeBreakdown,
		r.renderDistribution("by_status", report.Metrics.ByStatus, func(_ int, line string) string { return r.paintByStatus(line) }),
		r.renderAssignees,
		r.renderDistribution("by_type", report.Metrics.ByType, uncolored),
		r.renderDistribution("by_priority", report.Metrics.ByPriority, uncolored),
		r.renderRootCauses,
		r.renderTasks,
		r.renderNarrative,
	}
	for _, section := range sections {
		section(report)
	}
	_, err := io.WriteString(r.out, r.page.String())
	return err
}

func (r Renderer) renderHeader(report domain.Report) {
	people := r.t.T("scope_all_users")
	if len(report.Assignees) > 0 {
		people = strings.Join(report.Assignees, ", ")
	}
	r.printf("\n%s\n%s : %s → %s   %s : %s\n",
		r.t.T("report_title", report.Project),
		r.t.T("scope_period"), report.Period.FromString(), report.Period.ToString(),
		r.t.T("scope_users"), people)
}

func (r Renderer) renderOverview(report domain.Report) {
	m := report.Metrics
	r.section("overview")
	r.table(func(w io.Writer) {
		fmt.Fprintf(w, "%s%s\t%d\n", plainPrefix, r.t.T("total_tasks"), m.TotalTasks)
		fmt.Fprintf(w, "%s%s\t%d (%.0f%%)\n", plainPrefix, r.t.T("resolved"), m.ResolvedTasks, m.ResolutionRate()*percentPerUnit)
		fmt.Fprintf(w, "%s%s\t%s\n", plainPrefix, r.timeLoggedLabel(report), m.TotalTimeSpent)
		fmt.Fprintf(w, "%s%s\t%s\n", plainPrefix, r.t.T("avg_per_task"), m.AvgTimeSpent)
		fmt.Fprintf(w, "%s%s\t%s\n", plainPrefix, r.t.T("mttr"), r.formatMTTR(m.MTTR))
	})
}

// timeLoggedLabel names what the total actually covers. Targeting people filters
// twice — the tickets assigned to them, then their own worklogs — so the figure
// is their time *as assignee*, not every minute they logged that day.
func (r Renderer) timeLoggedLabel(report domain.Report) string {
	if len(report.Assignees) == 0 {
		return r.t.T("time_logged")
	}
	return r.t.T("time_logged_as_assignee")
}

// renderTimeScopeNote spells out what the assignee filter leaves out, since a
// missing worklog is invisible by definition.
func (r Renderer) renderTimeScopeNote(report domain.Report) {
	if len(report.Assignees) == 0 || report.Metrics.TotalTimeSpent.Seconds() == 0 {
		return
	}
	r.renderIndented(r.t.T("time_scope_note"))
}

// renderTimeBreakdown answers "where did the time go?": every ticket with time
// logged in the period, longest first, with its share of the total. Tickets
// touched without logged time are only counted — they carry no share.
func (r Renderer) renderTimeBreakdown(report domain.Report) {
	breakdown := report.Metrics.TimeBreakdown
	if len(breakdown) == 0 {
		return
	}
	r.section("time_breakdown")
	shown, hidden := window(breakdown, r.opts.TaskLimit)
	// A row carries no status text, so it is painted from its own ticket's
	// status: colouring by keywords would read the summary instead.
	paint := func(row int, line string) string {
		return r.paintWith(line, statusColor(shown[row].Status))
	}
	r.paintedTable(paint, func(w io.Writer) {
		for _, entry := range shown {
			fmt.Fprintf(w, "%s%s\t%s\t%3.0f%%\t%s\n",
				plainPrefix, entry.Key, entry.TimeSpent, entry.Share*percentPerUnit, r.clip(entry.Summary, breakdownSummaryLength))
		}
	})
	if hidden > 0 {
		r.printf("%s%s\n", plainPrefix, r.t.T("more_hidden", hidden))
	}
	r.renderUntimedCount(report.Metrics.UntimedTasks)
}

func (r Renderer) renderUntimedCount(untimed int) {
	if untimed == 0 {
		return
	}
	key := "untimed_tasks_many"
	if untimed == 1 {
		key = "untimed_tasks_one"
	}
	r.printf("%s%s\n", plainPrefix, r.t.T(key, untimed))
}

func (r Renderer) renderAssignees(report domain.Report) {
	// A single-person breakdown only repeats the overview totals.
	if len(report.Metrics.ByAssignee) <= 1 {
		return
	}
	r.section("by_assignee")
	r.table(func(w io.Writer) {
		for _, entry := range report.Metrics.ByAssignee {
			fmt.Fprintf(w, "%s%s\t%s\n", plainPrefix, entry.Assignee,
				r.t.T("assignee_line", entry.Total, entry.Resolved, entry.TimeSpent))
		}
	})
}

// uncolored leaves a distribution line as-is: only statuses carry a color.
func uncolored(_ int, line string) string { return line }

func (r Renderer) renderDistribution(titleKey string, counts []domain.LabelCount, paint rowPainter) func(domain.Report) {
	return func(_ domain.Report) {
		if len(counts) == 0 {
			return
		}
		r.section(titleKey)
		r.paintedTable(paint, func(w io.Writer) {
			for _, entry := range counts {
				fmt.Fprintf(w, "%s%s\t%d\n", plainPrefix, r.clip(entry.Label, detailTruncateLength), entry.Count)
			}
		})
	}
}

// renderRootCauses lists the causes shared by several tickets, each reduced to
// its gist and followed by how many tickets it hit. A cause hitting one ticket
// is already shown under that ticket, so it is left out here.
func (r Renderer) renderRootCauses(report domain.Report) {
	if r.opts.Detail.IsBrief() {
		return
	}
	var recurring []domain.LabelCount
	for _, entry := range report.Metrics.TopRootCauses {
		if entry.Count > 1 {
			recurring = append(recurring, entry)
		}
	}
	if len(recurring) == 0 {
		return
	}
	r.section("top_root_causes")
	for _, entry := range recurring {
		r.renderBullet(fmt.Sprintf("%s (×%d)", r.concise(entry.Label), entry.Count))
	}
}

func (r Renderer) renderTasks(report domain.Report) {
	if r.opts.Detail.IsBrief() {
		return
	}
	if len(report.Tasks) == 0 {
		r.section("no_tasks")
		return
	}
	shown, hidden := window(report.Tasks, r.opts.TaskLimit)
	r.section("tasks_header", len(shown), len(report.Tasks))
	for _, task := range shown {
		r.renderTask(task)
	}
	if hidden > 0 {
		r.printf("\n%s\n", r.t.T("more_hidden", hidden))
	}
}

// renderTask prints one ticket: a compact header line (key · status · time)
// followed by the summary, root cause and fix as word-wrapped indented blocks.
func (r Renderer) renderTask(task domain.Task) {
	// The whole block takes the status color, not just the header: a blocked
	// ticket must stand out down to its cause and fix lines.
	block := r.withTaskColor(task)
	block.printf("\n%s%s%s%s%s%s%s\n", bulletPrefix, task.Key, fieldSeparator, task.Status, fieldSeparator, task.TimeSpent, r.blockingMarker(task))
	block.renderIndented(task.Summary)
	if task.RootCause != "" {
		block.renderIndented(fmt.Sprintf("%s: %s", r.t.T("root_cause"), r.concise(task.RootCause)))
	}
	if task.Correction != "" {
		block.renderIndented(fmt.Sprintf("%s: %s", r.t.T("fix"), r.concise(task.Correction)))
	}
}

// blockingMarker justifies the red on a ticket whose status does not say
// blocked: the blocker was reported in the comments.
func (r Renderer) blockingMarker(task domain.Task) string {
	if !task.HasBlockingComment() || domain.ClassifyStatus(task.Status) == domain.StatusBlocked {
		return ""
	}
	return fieldSeparator + r.t.T("blocking_comment")
}

// renderBullet prints a word-wrapped block led by a bullet, wrapped lines
// hanging under the bullet text.
func (r Renderer) renderBullet(text string) {
	r.wrapLines(text, bulletPrefix, bulletContinuation)
}

// renderIndented prints a word-wrapped block indented under a ticket bullet.
func (r Renderer) renderIndented(text string) {
	r.wrapLines(text, bulletContinuation, bulletContinuation)
}

func (r Renderer) wrapLines(text, first, rest string) {
	width := r.wrapWidth() - utf8.RuneCountInString(first)
	for i, line := range wrapText(text, width) {
		prefix := rest
		if i == 0 {
			prefix = first
		}
		r.printf("%s%s\n", prefix, line)
	}
}

func (r Renderer) renderNarrative(report domain.Report) {
	narrative, ok := report.Narrative.Get()
	if !ok {
		return
	}
	r.section("ai_summary")
	// One line per ticket, each colored like its own ticket block. The opening
	// overview line names every status at once, so it stays neutral.
	colors := taskColors(report.Tasks)
	for _, line := range strings.Split(narrative, "\n") {
		r.printf("%s\n", r.paintNarrativeLine(line, colors))
	}
}

// window caps a list at limit entries and reports how many it left out. One
// rule for every list in the report: 0 (or less) means no cap.
func window[T any](items []T, limit int) ([]T, int) {
	if limit <= 0 || len(items) <= limit {
		return items, 0
	}
	return items[:limit], len(items) - limit
}

// section prints a translated title, set apart from the previous block by a
// blank line.
func (r Renderer) section(titleKey string, args ...any) {
	r.printf("\n\n%s\n", r.t.T(titleKey, args...))
}

func (r Renderer) printf(format string, args ...any) {
	text := fmt.Sprintf(format, args...)
	if r.lineColor != "" {
		text = colorizeLines(text, r.lineColor)
	}
	r.page.WriteString(text)
}

// rowPainter colours one already-aligned table row, identified by its index so
// a painter can look the row's own data up.
type rowPainter func(row int, line string) string

func (r Renderer) table(write func(io.Writer)) {
	r.paintedTable(uncolored, write)
}

// paintedTable aligns first, colors after: tabwriter counts escape codes as
// visible width, so they must be added once the padding is computed.
func (r Renderer) paintedTable(paint rowPainter, write func(io.Writer)) {
	var aligned strings.Builder
	w := tabwriter.NewWriter(&aligned, tableMinCellWidth, tableTabWidth, tablePadding, tablePadChar, tableFlags)
	write(w)
	// Flush can only fail on its destination's error, and a strings.Builder has none.
	_ = w.Flush()
	for row, line := range strings.Split(strings.TrimSuffix(aligned.String(), "\n"), "\n") {
		r.printf("%s\n", paint(row, line))
	}
}

func (r Renderer) formatMTTR(mttr domain.Optional[time.Duration]) string {
	duration, ok := mttr.Get()
	if !ok {
		return r.t.T("na")
	}
	days := int(duration.Hours()) / hoursPerDay
	hours := int(duration.Hours()) % hoursPerDay
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

// minWrapWidth guards against absurd widths, where wrapping would leave a
// word or two per line.
const minWrapWidth = 40

func (r Renderer) wrapWidth() int {
	if r.opts.Width < minWrapWidth {
		return defaultWrapWidth
	}
	return r.opts.Width
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
	if r.opts.Detail.IsFull() {
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
	if r.opts.Detail.IsFull() {
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
