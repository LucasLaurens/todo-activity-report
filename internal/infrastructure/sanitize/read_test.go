package sanitize_test

import (
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

func TestReadAtMostReturnsABodyWithinTheLimit(t *testing.T) {
	t.Parallel()
	// Act
	content, err := sanitize.ReadAtMost(strings.NewReader("abcd"), 4)

	// Assert
	if err != nil || string(content) != "abcd" {
		t.Fatalf("expected the whole body, got %q (%v)", content, err)
	}
}

func TestReadAtMostRefusesABodyOverTheLimit(t *testing.T) {
	t.Parallel()
	// Act
	_, err := sanitize.ReadAtMost(strings.NewReader("abcde"), 4)

	// Assert
	if err == nil {
		t.Fatal("expected a body one byte over the limit to be refused")
	}
}
