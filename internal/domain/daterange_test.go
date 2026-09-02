package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestParseDateRangeReadsBothBounds(t *testing.T) {
	t.Parallel()
	// Act
	period, err := domain.ParseDateRange("2026-05-01", "2026-05-31")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if period.FromString() != "2026-05-01" || period.ToString() != "2026-05-31" {
		t.Fatalf("unexpected bounds: %s → %s", period.FromString(), period.ToString())
	}
}

func TestParseDateRangeRejectsBadFromDate(t *testing.T) {
	t.Parallel()
	// Act
	_, err := domain.ParseDateRange("nope", "2026-05-31")

	// Assert
	if err == nil {
		t.Fatal("expected a parse error on the from date")
	}
}

func TestParseDateRangeRejectsBadToDate(t *testing.T) {
	t.Parallel()
	// Act
	_, err := domain.ParseDateRange("2026-05-01", "nope")

	// Assert
	if err == nil {
		t.Fatal("expected a parse error on the to date")
	}
}

func TestNewDateRangeRejectsInvertedRange(t *testing.T) {
	t.Parallel()
	// Arrange
	from := time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	// Act
	_, err := domain.NewDateRange(from, to)

	// Assert
	if !errors.Is(err, domain.ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestDateRangeExposesParsedTimes(t *testing.T) {
	t.Parallel()
	// Arrange
	from := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.May, 2, 0, 0, 0, 0, time.UTC)

	// Act
	period, err := domain.NewDateRange(from, to)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !period.From().Equal(from) || !period.To().Equal(to) {
		t.Fatalf("From/To not preserved: %v → %v", period.From(), period.To())
	}
}
