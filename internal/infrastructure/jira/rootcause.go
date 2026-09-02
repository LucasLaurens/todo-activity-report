package jira

import (
	"strings"
	"unicode"
)

// The root-cause custom field can hold a single structured note following the
// usual template (Problème / Cause / Correction / Prévention) rather than a
// bare root cause. We surface the *Cause* section as the task root cause and the
// *Correction* section as its fix, so a single Jira field feeds both columns.
//
// rootCauseTemplatePlaceholder is the instruction text left in an unfilled
// template. Its presence means the author never filled the note in, so the
// whole field is treated as empty rather than counted as a real root cause.
const rootCauseTemplatePlaceholder = "DU POINT DE VUE DU CLIENT"

const (
	categoryCause      = "cause"
	categoryCorrection = "correction"
	categoryIgnore     = "_ignore"
)

// sectionLabels maps the accent-stripped, lower-cased prefix of a template
// header to the section it opens. "correction" is checked before "cause" only
// for clarity; prefixes do not overlap.
var sectionLabels = []struct {
	prefix   string
	category string
}{
	{"correction", categoryCorrection},
	{"cause", categoryCause},
	{"probleme", categoryIgnore},
	{"prevention", categoryIgnore},
	{"recommandation", categoryIgnore},
}

// splitRootCause interprets that root-cause template, returning the
// technical Cause (root cause) and Correction (fix) on one clean line each.
// A plain, non-template value is returned verbatim as the root cause. An
// unfilled template yields two empty strings.
func splitRootCause(text string) (rootCause, correction string) {
	text = strings.TrimSpace(text)
	if text == "" || strings.Contains(text, rootCauseTemplatePlaceholder) {
		return "", ""
	}

	sections := make(map[string][]string)
	current := ""
	headerSeen := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if category, rest, ok := matchSection(line); ok {
			headerSeen = true
			current = category
			if rest != "" {
				sections[current] = append(sections[current], rest)
			}
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}

	// No template headers at all → a plain free-text field; keep it as the cause.
	if !headerSeen {
		return text, ""
	}
	return flattenSection(sections[categoryCause]), flattenSection(sections[categoryCorrection])
}

// matchSection reports whether a line opens a template section. "Cause
// vulgarisée" / "Correction vulgarisée" headers are routed to categoryIgnore so
// the lay-friendly duplicate does not pollute the technical section.
func matchSection(line string) (category, rest string, ok bool) {
	head := stripLeadingNonLetters(line)
	normalized := normalizeText(head)
	for _, label := range sectionLabels {
		if !strings.HasPrefix(normalized, label.prefix) {
			continue
		}
		category = label.category
		if strings.Contains(normalized, "vulgaris") {
			category = categoryIgnore
		}
		return category, inlineRemainder(head), true
	}
	return "", "", false
}

// inlineRemainder returns the body following "Label :" when header and content
// share a line (the unfilled-template layout); empty when the header stands alone.
func inlineRemainder(head string) string {
	_, after, found := strings.Cut(head, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(after)
}

func stripLeadingNonLetters(s string) string {
	return strings.TrimLeftFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
}

var accentReplacer = strings.NewReplacer(
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"à", "a", "â", "a", "ä", "a",
	"î", "i", "ï", "i",
	"ô", "o", "ö", "o",
	"û", "u", "ü", "u", "ù", "u",
	"ç", "c",
)

func normalizeText(s string) string {
	return accentReplacer.Replace(strings.ToLower(s))
}

// flattenSection joins a section's lines into a single space-collapsed line, so
// it renders on one row and aggregates cleanly in the root-cause distribution.
func flattenSection(lines []string) string {
	return strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
}
