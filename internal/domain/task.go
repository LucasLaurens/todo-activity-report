package domain

import (
	"strings"
	"time"
)

// Task is a ticket processed during the reported period, whatever its status:
// time logged on it or its status moved (or, with --touched, any update).
type Task struct {
	Key        string
	Summary    string
	Assignee   string
	Type       string
	Priority   string
	Status     string
	RootCause  string
	Correction string
	CreatedAt  time.Time
	ResolvedAt Optional[time.Time]
	TimeSpent  TimeSpent
	Comments   []Comment
}

// Comment is a discussion entry attached to a ticket, kept as plain text so the
// summarizer can lean on it for accurate, detailed recaps without reaching into
// the tracker's rich-text representation.
type Comment struct {
	Author string
	Body   string
}

// IsResolvedWithin reports whether the ticket was resolved *during* the period.
// A ticket resolved months ago and merely touched today is not a resolution of
// today: counting it would inflate the resolution rate and the MTTR of a
// one-day report.
func (t Task) IsResolvedWithin(period DateRange) bool {
	resolvedAt, ok := t.ResolvedAt.Get()
	return ok && period.Contains(resolvedAt)
}

// ResolutionDuration is the wall-clock time from creation to resolution. A
// resolution dated before creation (cloned/imported issues) is a data anomaly
// and yields no duration rather than a negative one.
func (t Task) ResolutionDuration() Optional[time.Duration] {
	resolvedAt, ok := t.ResolvedAt.Get()
	if !ok || !resolvedAt.After(t.CreatedAt) {
		return None[time.Duration]()
	}
	return Some(resolvedAt.Sub(t.CreatedAt))
}

func (t Task) HasRootCause() bool {
	return t.RootCause != ""
}

// blockingCommentKeywords flag a comment that reports a blocker, so a ticket
// can read as blocked even when its status does not say so (« In progress dev »
// with « bloqué par l'infra » in the thread).
var blockingCommentKeywords = []string{"bloqu", "blocked", "blocker", "blocage", "impedi", "en attente", "waiting for", "stuck"}

// HasBlockingComment reports whether any comment the tracker returned for the
// ticket mentions a blocker.
func (t Task) HasBlockingComment() bool {
	for _, comment := range t.Comments {
		lowered := strings.ToLower(comment.Body)
		for _, keyword := range blockingCommentKeywords {
			if strings.Contains(lowered, keyword) {
				return true
			}
		}
	}
	return false
}

// StatusKind is what this ticket's status means for the reader. A blocker
// reported in the comments outranks the status.
func (t Task) StatusKind() StatusKind {
	if t.HasBlockingComment() {
		return StatusBlocked
	}
	return ClassifyStatus(t.Status)
}
