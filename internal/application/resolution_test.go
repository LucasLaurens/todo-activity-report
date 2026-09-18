package application_test

import (
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func singleDay(t *testing.T, day time.Time) domain.DateRange {
	t.Helper()
	period, err := domain.NewDateRange(day, day)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return period
}

// A ticket resolved long ago and merely touched today is not a resolution of
// today: counting it inflated both the resolution rate and the MTTR.
func TestCalculateCountsOnlyResolutionsInsideThePeriod(t *testing.T) {
	t.Parallel()
	// Arrange — one ticket resolved today, one resolved two months ago.
	today := time.Date(2026, time.May, 10, 0, 0, 0, 0, time.UTC)
	resolvedToday := testsupport.NewTask().WithKey("HT-1").
		CreatedOn(today.Add(9 * time.Hour)).ResolvedOn(today.Add(17 * time.Hour)).Build()
	resolvedLongAgo := testsupport.NewTask().WithKey("HT-2").
		CreatedOn(today.AddDate(0, -3, 0)).ResolvedOn(today.AddDate(0, -2, 0)).Build()

	// Act
	metrics := application.NewMetricsCalculator().Calculate([]domain.Task{resolvedToday, resolvedLongAgo}, singleDay(t, today))

	// Assert
	if metrics.ResolvedTasks != 1 {
		t.Fatalf("expected 1 resolution in the period, got %d", metrics.ResolvedTasks)
	}
	if metrics.ByAssignee[0].Resolved != 1 {
		t.Fatalf("expected the per-person count to match, got %d", metrics.ByAssignee[0].Resolved)
	}
	duration, present := metrics.MTTR.Get()
	if !present {
		t.Fatal("expected an MTTR for the ticket resolved today")
	}
	if duration != 8*time.Hour {
		t.Fatalf("expected the MTTR of the in-period resolution only, got %s", duration)
	}
}
