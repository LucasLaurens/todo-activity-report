package sanitize_test

import (
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

func TestPrintableRemovesAnsiSequencesAndControlCharacters(t *testing.T) {
	t.Parallel()
	// Arrange — a ticket summary crafted to repaint and erase the terminal.
	hostile := "normal \x1b[2J\x1b[31mred\x1b[0m\x07 tail"

	// Act
	cleaned := sanitize.Printable(hostile)

	// Assert
	if cleaned != "normal red tail" {
		t.Fatalf("unexpected sanitisation: %q", cleaned)
	}
}

func TestPrintableDropsCarriageReturnsUsedToOverwriteALine(t *testing.T) {
	t.Parallel()
	// Arrange
	hostile := "visible\rhidden"

	// Act
	cleaned := sanitize.Printable(hostile)

	// Assert
	if strings.ContainsRune(cleaned, '\r') {
		t.Fatalf("carriage return survived: %q", cleaned)
	}
}

func TestPrintableKeepsNewlinesTabsAndAccents(t *testing.T) {
	t.Parallel()
	// Arrange
	legit := "Cause : dérive\n\tCorrection : épinglée"

	// Act
	cleaned := sanitize.Printable(legit)

	// Assert
	if cleaned != legit {
		t.Fatalf("legit text altered: %q", cleaned)
	}
}
