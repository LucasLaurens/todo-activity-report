package domain_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestQueryHasAssigneeFilter(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if (domain.Query{}).HasAssigneeFilter() {
		t.Fatal("expected no filter for empty assignees")
	}
	if !(domain.Query{Assignees: []string{"u1"}}).HasAssigneeFilter() {
		t.Fatal("expected filter when assignees present")
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
