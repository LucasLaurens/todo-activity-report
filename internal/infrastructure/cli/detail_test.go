package cli_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/cli"
)

func TestParseDetailLevelReadsTheThreeLevels(t *testing.T) {
	t.Parallel()
	// Arrange
	cases := map[string]cli.DetailLevel{
		"":       cli.DetailNormal,
		"normal": cli.DetailNormal,
		"brief":  cli.DetailBrief,
		"FULL":   cli.DetailFull,
	}

	for value, expected := range cases {
		t.Run(value, func(t *testing.T) {
			t.Parallel()
			// Act
			level, err := cli.ParseDetailLevel(value)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if level != expected {
				t.Errorf("expected %v, got %v", expected, level)
			}
		})
	}
}

func TestParseDetailLevelRejectsAnUnknownLevel(t *testing.T) {
	t.Parallel()
	// Act
	_, err := cli.ParseDetailLevel("verbose")

	// Assert
	if err == nil {
		t.Fatal("expected an error for an unknown detail level")
	}
}
