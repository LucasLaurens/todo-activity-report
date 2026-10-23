package domain

import "strings"

// StatusKind is what a tracker status means for the reader of the report: what
// needs attention, what is moving, what is delivered, what is noise.
type StatusKind int

const (
	// StatusNotStarted is the default: queued work, nothing to report yet.
	StatusNotStarted StatusKind = iota
	StatusBlocked
	StatusInFlight
	StatusDone
	StatusAbandoned
)

// statusRules recognises a status by keyword: tracker status names are
// free-form and localised (fr/en on the boards this was built against), so an
// exhaustive enum would break on the next workflow edit.
//
// ORDER MATTERS — first match wins, specific before generic. The three
// "waiting" statuses of the board mean three different things:
// « Waiting for release » is delivered work waiting for the train,
// « Waiting for feat env » is work in flight waiting for its environment, and
// « En attente retour client » is a blocking wait, with the other waits.
var statusRules = []struct {
	kind     StatusKind
	keywords []string
}{
	{StatusDone, []string{"release"}},
	{StatusInFlight, []string{"feat env"}},
	{StatusAbandoned, []string{"annul", "cancel", "obsol"}},
	{StatusBlocked, []string{"block", "bloqu", "impedi", "hold", "stuck", "attente", "waiting", "reject", "rejet", "fail", "echec", "échec", "error", "erreur"}},
	{StatusDone, []string{"deploy", "complet", "prod", "termin", "done", "ended", "clos", "ferm", "resolv", "résolu", "resolu", "livr", "merged", "valid"}},
	{StatusInFlight, []string{"progress", "cours", "doing", "dev", "review", "relecture", "test", "recette", "qa", "verif", "vérif"}},
}

// ClassifyStatus maps a tracker status name to what it means for the reader.
func ClassifyStatus(status string) StatusKind {
	lowered := strings.ToLower(status)
	for _, rule := range statusRules {
		for _, keyword := range rule.keywords {
			if strings.Contains(lowered, keyword) {
				return rule.kind
			}
		}
	}
	return StatusNotStarted
}

// attentionOrder lists the kinds as the report reads them: what blocks first,
// then what is moving, then what shipped, then what is not started, and last
// what was abandoned.
var attentionOrder = []StatusKind{StatusBlocked, StatusInFlight, StatusDone, StatusNotStarted, StatusAbandoned}

// AttentionRank is the sort key that puts the blocking subjects first. A kind
// missing from attentionOrder sinks to the bottom rather than posing as a
// blocker. O(k) for k kinds, a handful.
func (k StatusKind) AttentionRank() int {
	for rank, kind := range attentionOrder {
		if kind == k {
			return rank
		}
	}
	return len(attentionOrder)
}
