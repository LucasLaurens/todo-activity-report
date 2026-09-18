package cli

import (
	"fmt"
	"strings"
)

// DetailLevel is how much of the report is printed. One knob instead of two
// booleans: "brief" and "full" are two points on the same axis, and a flag pair
// would allow the contradictory --brief --full.
type DetailLevel int

const (
	// DetailNormal is the default: figures, root causes and the ticket list,
	// with long texts reduced to their gist.
	DetailNormal DetailLevel = iota
	// DetailBrief keeps the figures and the AI briefing only.
	DetailBrief
	// DetailFull prints every text in full.
	DetailFull
)

var detailLevels = map[string]DetailLevel{
	"":       DetailNormal,
	"normal": DetailNormal,
	"brief":  DetailBrief,
	"full":   DetailFull,
}

// DetailLevelNames lists the accepted values, for the help text and errors.
func DetailLevelNames() string {
	return "brief|normal|full"
}

func ParseDetailLevel(value string) (DetailLevel, error) {
	level, ok := detailLevels[strings.ToLower(strings.TrimSpace(value))]
	if !ok {
		return DetailNormal, fmt.Errorf("invalid --detail %q, expected %s", value, DetailLevelNames())
	}
	return level, nil
}

// IsBrief reports whether only the figures and the AI briefing are printed.
func (d DetailLevel) IsBrief() bool { return d == DetailBrief }

// IsFull reports whether texts are printed untruncated.
func (d DetailLevel) IsFull() bool { return d == DetailFull }
