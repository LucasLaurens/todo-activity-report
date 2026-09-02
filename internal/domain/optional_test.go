package domain_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestSomeIsPresentAndReturnsValue(t *testing.T) {
	t.Parallel()
	// Arrange
	optional := domain.Some("hello")

	// Act
	value, present := optional.Get()

	// Assert
	if !present || value != "hello" {
		t.Fatalf("expected present \"hello\", got present=%v value=%q", present, value)
	}
}

func TestSomeOrElseReturnsValue(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if got := domain.Some("x").OrElse("fallback"); got != "x" {
		t.Fatalf("expected \"x\", got %q", got)
	}
}

func TestNoneGetReportsAbsent(t *testing.T) {
	t.Parallel()
	// Act
	_, present := domain.None[int]().Get()

	// Assert
	if present {
		t.Fatal("expected absent")
	}
}

func TestNoneIsAbsentAndUsesFallback(t *testing.T) {
	t.Parallel()
	// Arrange
	optional := domain.None[string]()

	// Act
	value := optional.OrElse("fallback")

	// Assert
	if optional.IsPresent() {
		t.Fatal("expected None to be absent")
	}
	if value != "fallback" {
		t.Fatalf("expected \"fallback\", got %q", value)
	}
}
