package domain

import (
	"strings"
	"time"
)

// Task is a resolved unit of work pulled from the tracker.
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

func (t Task) IsResolved() bool {
	return t.ResolvedAt.IsPresent()
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

func (t Task) HasComments() bool {
	return len(t.Comments) > 0
}

// blockingCommentKeywords flag a comment that reports a blocker, so a ticket
// can read as blocked even when its status does not say so (« In progress dev »
// with « bloqué par l'infra » in the thread).
var blockingCommentKeywords = []string{"bloqu", "blocked", "blocker", "blocage", "impedi", "en attente", "waiting for", "stuck"}

// HasBlockingComment reports whether a recent comment mentions a blocker.
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
