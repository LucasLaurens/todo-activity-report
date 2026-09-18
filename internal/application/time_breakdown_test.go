package application_test

import (
	"math"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func TestCalculateRanksTimeByTicketWithItsShareOfTheDay(t *testing.T) {
	t.Parallel()
	// Arrange — three hours logged in total, one ticket touched without any.
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-2").WithTimeSpentHours(1).Build(),
		testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(2).Build(),
		testsupport.NewTask().WithKey("HT-3").WithTimeSpentHours(0).Build(),
	}

	// Act
	metrics := application.NewMetricsCalculator().Calculate(tasks, wholeMay(t))

	// Assert — longest first, untimed tickets counted apart.
	breakdown := metrics.TimeBreakdown
	if len(breakdown) != 2 {
		t.Fatalf("expected the 2 timed tickets, got %+v", breakdown)
	}
	if breakdown[0].Key != "HT-1" || breakdown[1].Key != "HT-2" {
		t.Fatalf("expected HT-1 then HT-2, got %+v", breakdown)
	}
	if math.Abs(breakdown[0].Share-2.0/3.0) > 0.001 {
		t.Fatalf("expected a two-thirds share, got %v", breakdown[0].Share)
	}
	if metrics.UntimedTasks != 1 {
		t.Fatalf("expected 1 untimed ticket, got %d", metrics.UntimedTasks)
	}
}

func TestCalculateBreaksTimeTiesByTicketKey(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-9").WithTimeSpentHours(1).Build(),
		testsupport.NewTask().WithKey("HT-4").WithTimeSpentHours(1).Build(),
	}

	// Act
	metrics := application.NewMetricsCalculator().Calculate(tasks, wholeMay(t))

	// Assert
	if metrics.TimeBreakdown[0].Key != "HT-4" {
		t.Fatalf("expected the tie broken by key, got %+v", metrics.TimeBreakdown)
	}
}
