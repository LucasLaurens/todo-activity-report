// Package sanitize keeps data coming from outside the CLI safe to handle and to
// print: a ticket summary, a comment or a model answer can carry escape
// sequences that repaint, erase or spoof the report, and a response body can
// be arbitrarily large.
package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

// ansiSequence matches the escape sequences a terminal interprets: CSI colours
// and cursor moves, OSC strings, and the shorter two-byte escapes.
var ansiSequence = regexp.MustCompile(`\x1b(\[[0-?]*[ -/]*[@-~]|\][^\a\x1b]*(\a|\x1b\\)|[@-Z\\-_])`)

// Printable strips terminal escape sequences, control characters (C0, DEL and
// the C1 range some terminals still honour) and bidirectional overrides that
// can make text read differently from what it is. Newlines and tabs, the
// renderer's own layout, and every printable rune are kept.
func Printable(value string) string {
	stripped := ansiSequence.ReplaceAllString(value, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, stripped)
}
