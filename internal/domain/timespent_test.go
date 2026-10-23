package domain_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestNewTimeSpentClampsNegativeToZero(t *testing.T) {
	t.Parallel()
	// Act
	result := domain.NewTimeSpent(-100)

	// Assert
	if result.Seconds() != 0 {
		t.Fatalf("expected 0 seconds, got %d", result.Seconds())
	}
}

func TestTimeSpentAddIsCumulative(t *testing.T) {
	t.Parallel()
	// Arrange
	first := domain.NewTimeSpent(3600)
	second := domain.NewTimeSpent(1800)

	// Act
	total := first.Add(second)

	// Assert
	if total.Seconds() != 5400 {
		t.Fatalf("expected 5400 seconds, got %d", total.Seconds())
	}
}

func TestTimeSpentDividedByZeroReturnsZero(t *testing.T) {
	t.Parallel()
	// Arrange
	value := domain.NewTimeSpent(3600)

	// Act
	result := value.DividedBy(0)

	// Assert
	if result.Seconds() != 0 {
		t.Fatalf("expected 0 seconds, got %d", result.Seconds())
	}
}

func TestTimeSpentHoursAndSeconds(t *testing.T) {
	t.Parallel()
	// Arrange
	value := domain.NewTimeSpent(5400)

	// Act & Assert
	if value.Seconds() != 5400 {
		t.Fatalf("expected 5400 seconds, got %d", value.Seconds())
	}
	if value.Hours() != 1.5 {
		t.Fatalf("expected 1.5 hours, got %v", value.Hours())
	}
}

func TestTimeSpentIsZeroWithoutLoggedTime(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if !domain.NewTimeSpent(0).IsZero() {
		t.Fatal("expected no logged time to be zero")
	}
}

func TestTimeSpentIsNotZeroWithLoggedTime(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if domain.NewTimeSpent(1).IsZero() {
		t.Fatal("expected one second to be non-zero")
	}
}

func TestTimeSpentDividedByPositive(t *testing.T) {
	t.Parallel()
	// Act
	result := domain.NewTimeSpent(3600).DividedBy(2)

	// Assert
	if result.Seconds() != 1800 {
		t.Fatalf("expected 1800 seconds, got %d", result.Seconds())
	}
}

func TestTimeSpentStringMinutesOnly(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if got := domain.NewTimeSpent(45 * 60).String(); got != "45m" {
		t.Fatalf("expected \"45m\", got %q", got)
	}
}

func TestTimeSpentStringFormatsHoursAndMinutes(t *testing.T) {
	t.Parallel()
	// Arrange
	value := domain.NewTimeSpent(2*3600 + 5*60)

	// Act
	formatted := value.String()

	// Assert
	if formatted != "2h 05m" {
		t.Fatalf("expected \"2h 05m\", got %q", formatted)
	}
}

func TestShareOfReturnsTheFractionOfTheTotal(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if share := domain.NewTimeSpent(3600).ShareOf(domain.NewTimeSpent(14400)); share != 0.25 {
		t.Fatalf("expected a quarter, got %v", share)
	}
}

func TestShareOfAnEmptyTotalIsZero(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if share := domain.NewTimeSpent(3600).ShareOf(domain.NewTimeSpent(0)); share != 0 {
		t.Fatalf("expected zero, got %v", share)
	}
}
