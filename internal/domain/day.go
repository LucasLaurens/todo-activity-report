package domain

import (
	"fmt"
	"strings"
	"time"
)

// dayLayout is the date format the CLI speaks, matching how the team writes
// dates in Jira and in chat.
const dayLayout = "02/01/2006"

// relativeDays are the shortcuts accepted instead of a date, expressed as a
// number of days before today. "the day before" is what the report is asked for
// nine times out of ten, so it must be sayable.
var relativeDays = map[string]int{
	"today":       0,
	"aujourdhui":  0,
	"aujourd'hui": 0,
	"ce jour":     0,
	"yesterday":   1,
	"hier":        1,
}

// ParseDay turns a CLI date into the day to report on, relative to now:
// a DD/MM/YYYY date, a keyword ("hier", "yesterday", "today"), or — when empty —
// the last working day, so a bare run on Monday reports on Friday.
func ParseDay(value string, now time.Time) (time.Time, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return lastWorkingDay(now), nil
	}
	if daysBefore, ok := relativeDays[trimmed]; ok {
		return startOfDay(now).AddDate(0, 0, -daysBefore), nil
	}
	parsed, err := time.ParseInLocation(dayLayout, trimmed, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected DD/MM/YYYY, 'hier' or 'today'", value)
	}
	return parsed, nil
}

// lastWorkingDay is the previous day, skipping back over the weekend: Monday
// reports on Friday, and a run on Saturday or Sunday still reports on Friday.
func lastWorkingDay(now time.Time) time.Time {
	day := startOfDay(now).AddDate(0, 0, -1)
	for day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
		day = day.AddDate(0, 0, -1)
	}
	return day
}

func startOfDay(moment time.Time) time.Time {
	year, month, dayOfMonth := moment.Date()
	return time.Date(year, month, dayOfMonth, 0, 0, 0, 0, moment.Location())
}
