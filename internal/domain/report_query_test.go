package domain_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestHasAssigneeFilter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		assignees []string
		expected  bool
	}{
		{name: "whole board", assignees: nil, expected: false},
		{name: "chosen people", assignees: []string{"u1"}, expected: true},
	}
	for _, tc := range cases {
		t.Run("query/"+tc.name, func(t *testing.T) {
			t.Parallel()
			// Act
			filtered := domain.Query{Assignees: tc.assignees}.HasAssigneeFilter()

			// Assert
			if filtered != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, filtered)
			}
		})
		t.Run("report/"+tc.name, func(t *testing.T) {
			t.Parallel()
			// Act
			filtered := domain.Report{Assignees: tc.assignees}.HasAssigneeFilter()

			// Assert
			if filtered != tc.expected {
				t.Fatalf("expected %v, got %v", tc.expected, filtered)
			}
		})
	}
}

func TestMetricsResolutionRate(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if (domain.Metrics{}).ResolutionRate() != 0 {
		t.Fatal("expected 0 rate for empty metrics")
	}
	half := domain.Metrics{TotalTasks: 4, ResolvedTasks: 2}
	if half.ResolutionRate() != 0.5 {
		t.Fatalf("expected 0.5, got %v", half.ResolutionRate())
	}
}
