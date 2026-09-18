// Package sanitize keeps text coming from outside the CLI safe to print on a
// terminal: a ticket summary, a comment or a model answer can carry escape
// sequences that repaint, erase or spoof the report.
package sanitize

import (
	"regexp"
	"strings"
)

// ansiSequence matches the escape sequences a terminal interprets: CSI colours
// and cursor moves, OSC strings, and the shorter two-byte escapes.
var ansiSequence = regexp.MustCompile(`\x1b(\[[0-?]*[ -/]*[@-~]|\][^\a\x1b]*(\a|\x1b\\)|[@-Z\\-_])`)

// Printable strips terminal escape sequences and control characters, keeping
// newlines and tabs (the renderer's own layout) and every printable rune.
func Printable(value string) string {
	stripped := ansiSequence.ReplaceAllString(value, "")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, stripped)
}
