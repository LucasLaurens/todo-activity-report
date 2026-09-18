package domain_test

import (
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func may(dayOfMonth, hour int) time.Time {
	return time.Date(2026, time.May, dayOfMonth, hour, 0, 0, 0, time.UTC)
}

func mayRange(t *testing.T, from, to int) domain.DateRange {
	t.Helper()
	period, err := domain.NewDateRange(may(from, 0), may(to, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return period
}

func TestContainsCoversTheWholeLastDay(t *testing.T) {
	t.Parallel()
	// Arrange — an inclusive range runs to 23:59 of its last day.
	period := mayRange(t, 10, 12)

	// Act & Assert
	cases := map[time.Time]bool{
		may(9, 23):  false,
		may(10, 0):  true,
		may(12, 23): true,
		may(13, 0):  false,
	}
	for moment, expected := range cases {
		if period.Contains(moment) != expected {
			t.Fatalf("Contains(%s) = %v, want %v", moment, !expected, expected)
		}
	}
}

func TestIsResolvedWithinIgnoresAResolutionOlderThanThePeriod(t *testing.T) {
	t.Parallel()
	// Arrange — the ticket was resolved in March and only touched in May, so it
	// is not a resolution of this period.
	period := mayRange(t, 10, 10)
	old := domain.Task{
		CreatedAt:  time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC),
		ResolvedAt: domain.Some(time.Date(2026, time.March, 2, 9, 0, 0, 0, time.UTC)),
	}
	today := domain.Task{CreatedAt: may(10, 9), ResolvedAt: domain.Some(may(10, 17))}
	never := domain.Task{CreatedAt: may(10, 9), ResolvedAt: domain.None[time.Time]()}

	// Act & Assert
	if old.IsResolvedWithin(period) {
		t.Fatal("a resolution predating the period must not count")
	}
	if !today.IsResolvedWithin(period) {
		t.Fatal("a resolution inside the period must count")
	}
	if never.IsResolvedWithin(period) {
		t.Fatal("an unresolved ticket must not count")
	}
}
