package domain_test

import (
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func day(year int, month time.Month, dayOfMonth int) time.Time {
	return time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.Local)
}

func TestParseDayReadsTheFrenchFormat(t *testing.T) {
	t.Parallel()
	// Arrange
	now := day(2026, time.June, 10).Add(15 * time.Hour)

	// Act
	parsed, err := domain.ParseDay("08/06/2026", now)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parsed.Equal(day(2026, time.June, 8)) {
		t.Fatalf("expected 2026-06-08, got %s", parsed)
	}
}

func TestParseDayAcceptsRelativeKeywords(t *testing.T) {
	t.Parallel()
	// Arrange — Wednesday 10 June 2026, mid-afternoon.
	now := day(2026, time.June, 10).Add(15 * time.Hour)
	cases := map[string]time.Time{
		"hier":         day(2026, time.June, 9),
		"yesterday":    day(2026, time.June, 9),
		"aujourd'hui":  day(2026, time.June, 10),
		"today":        day(2026, time.June, 10),
		"  Yesterday ": day(2026, time.June, 9),
	}

	for value, expected := range cases {
		// Act
		parsed, err := domain.ParseDay(value, now)

		// Assert
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", value, err)
		}
		if !parsed.Equal(expected) {
			t.Fatalf("%q: expected %s, got %s", value, expected, parsed)
		}
	}
}

func TestParseDayWithoutValueFallsBackToTheLastWorkingDay(t *testing.T) {
	t.Parallel()
	// Arrange — 8 June 2026 is a Monday, 6 June a Saturday, 7 June a Sunday.
	cases := map[time.Time]time.Time{
		day(2026, time.June, 10): day(2026, time.June, 9), // Wednesday → Tuesday
		day(2026, time.June, 8):  day(2026, time.June, 5), // Monday → Friday
		day(2026, time.June, 7):  day(2026, time.June, 5), // Sunday → Friday
		day(2026, time.June, 6):  day(2026, time.June, 5), // Saturday → Friday
		day(2026, time.June, 9):  day(2026, time.June, 8), // Tuesday → Monday
	}

	for now, expected := range cases {
		// Act
		parsed, err := domain.ParseDay("", now)

		// Assert
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", now.Weekday(), err)
		}
		if !parsed.Equal(expected) {
			t.Fatalf("%s: expected %s, got %s", now.Weekday(), expected, parsed)
		}
	}
}

func TestParseDayRejectsAnUnknownValue(t *testing.T) {
	t.Parallel()
	// Arrange
	now := day(2026, time.June, 10)

	// Act
	_, err := domain.ParseDay("2026-06-08", now)

	// Assert
	if err == nil {
		t.Fatal("expected an error for a non DD/MM/YYYY value")
	}
}
