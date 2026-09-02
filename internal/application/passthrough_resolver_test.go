package application_test

import (
	"context"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
)

func TestPassthroughResolverReturnsTermUnchanged(t *testing.T) {
	t.Parallel()
	// Arrange
	term := "Ada Lovelace"

	// Act
	got, err := application.PassthroughResolver{}.Resolve(context.Background(), term)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != term {
		t.Fatalf("expected %q, got %q", term, got)
	}
}
