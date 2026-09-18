package llm

import (
	"fmt"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// maxTasksInPrompt and maxCommentsInPrompt budget the request: the model writes
// a line per ticket and reads every comment first, so these two numbers drive
// how long the narrative takes. A day rarely holds twenty worked tickets, and
// the three most recent comments carry the state of a thread.
const (
	maxTasksInPrompt    = 20
	maxCommentsInPrompt = 3
)

// maxCommentRunes bounds a single comment in the prompt: enough to carry the
// gist, capped so a long thread can't blow up the request size (and cost).
const maxCommentRunes = 280

// Length budget for the spoken briefing: the recap is read aloud, so a line
// per ticket must stay one breath long.
const (
	maxOverviewSentences  = 2
	maxWordsPerTicketLine = 20
)

// oneLine collapses a comment to a single whitespace-normalised line, clipped to
// maxCommentRunes so the model gets the gist without the full thread.
func oneLine(text string) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	runes := []rune(collapsed)
	if len(runes) <= maxCommentRunes {
		return collapsed
	}
	return string(runes[:maxCommentRunes-1]) + "…"
}

// recentComments keeps the tail of the thread — the mapper hands comments over
// oldest first, and the recent ones say where the ticket stands.
func recentComments(comments []domain.Comment) []domain.Comment {
	if len(comments) <= maxCommentsInPrompt {
		return comments
	}
	return comments[len(comments)-maxCommentsInPrompt:]
}

// buildPrompt turns the report into a compact briefing for the model. The
// deterministic metrics are passed in so the model narrates rather than
// re-computes them. The translator sets the response language.
func buildPrompt(report domain.Report, t i18n.Translator) string {
	var b strings.Builder

	language := t.T("prompt_language")
	b.WriteString(language + " ")
	b.WriteString("You are an engineering delivery analyst. Produce a briefing the engineer can read ALOUD as a spoken recap.\n")
	b.WriteString("Structure it in two parts:\n")
	fmt.Fprintf(&b, "1. %d sentences of overview at most: volume, time spent, status mix and the main risk.\n", maxOverviewSentences)
	b.WriteString("2. Then give a progress recap for EVERY ticket — one line each, starting with the ticket key. ")
	b.WriteString("KEEP THE TICKET ORDER GIVEN BELOW: it already reads blockers first, then work in flight, then what shipped, ")
	b.WriteString("then what is not started, then what was abandoned. Do not reorder, do not re-group. ")
	b.WriteString("Every ticket gets its own progress line, with NO exception: do not omit any ticket, do not merge several into one line, ")
	b.WriteString("and NEVER group tickets into a catch-all line such as \"no documented root cause\". A missing root cause is not a reason to skip a ticket.\n")
	b.WriteString("   - Done/resolved tickets: state the root cause plainly in one sentence, then the fix in a brief clause.\n")
	b.WriteString("   - Tickets still open (in progress, blocked, in code review, in release): state what was advanced and any blocker or next step. ")
	b.WriteString("When no root cause, fix or comment is recorded, still write a progress line inferred from the ticket summary, status and time logged ")
	b.WriteString("(e.g. work underway, awaiting review, blocked, pending release).\n")
	fmt.Fprintf(&b, "Length budget, and it is hard: %d words maximum per ticket line, one single line each, no sub-bullets, "+
		"no blank line between them, no preamble, no closing sentence, no markdown, no heading. Be terse: drop filler, keep the fact.\n",
		maxWordsPerTicketLine)
	b.WriteString("The ticket list may end with \"... and N more\": those tickets are out of scope, ignore them silently — ")
	b.WriteString("do not mention them, do not count them, do not close with a sentence about them.\n")
	b.WriteString("Each ticket may list recent comments. Use them as supporting detail to make the recap precise and accurate — ")
	b.WriteString("fold the relevant fact into that ticket's line. Do not quote comments verbatim, do not list them, and do not invent ")
	b.WriteString("anything not present in the data below.\n\n")

	metrics := report.Metrics
	fmt.Fprintf(&b, "Project: %s\n", report.Project)
	fmt.Fprintf(&b, "Period: %s to %s\n", report.Period.FromString(), report.Period.ToString())
	fmt.Fprintf(&b, "Total tickets: %d (resolved %d)\n", metrics.TotalTasks, metrics.ResolvedTasks)
	fmt.Fprintf(&b, "Time logged as assignee: %s, average %s per ticket\n", metrics.TotalTimeSpent, metrics.AvgTimeSpent)

	if len(metrics.ByStatus) > 0 {
		b.WriteString("Status breakdown:\n")
		for _, status := range metrics.ByStatus {
			fmt.Fprintf(&b, "  - %s (%d)\n", status.Label, status.Count)
		}
	}
	if len(metrics.TopRootCauses) > 0 {
		b.WriteString("Top root causes:\n")
		for _, cause := range metrics.TopRootCauses {
			fmt.Fprintf(&b, "  - %s (%d)\n", cause.Label, cause.Count)
		}
	}

	b.WriteString("\nTickets:\n")
	for index, task := range report.Tasks {
		if index >= maxTasksInPrompt {
			fmt.Fprintf(&b, "  ... and %d more\n", len(report.Tasks)-maxTasksInPrompt)
			break
		}
		fmt.Fprintf(&b, "  - [%s] %s | status=%s | assignee=%s | time=%s | root cause=%q | fix=%q\n",
			task.Key, task.Summary, task.Status, task.Assignee, task.TimeSpent, task.RootCause, task.Correction)
		for _, comment := range recentComments(task.Comments) {
			fmt.Fprintf(&b, "      comment by %s: %s\n", comment.Author, oneLine(comment.Body))
		}
	}

	b.WriteString("\n" + language)
	return b.String()
}
