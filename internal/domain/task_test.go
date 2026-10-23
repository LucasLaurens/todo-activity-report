package domain_test

import (
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func baseTask() domain.Task {
	created := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)
	return domain.Task{
		Key:        "HT-1",
		CreatedAt:  created,
		ResolvedAt: domain.Some(created.Add(3 * time.Hour)),
		RootCause:  "Config drift",
		TimeSpent:  domain.NewTimeSpent(3600),
	}
}

func TestResolutionDurationComputesDelta(t *testing.T) {
	t.Parallel()
	// Act
	duration, ok := baseTask().ResolutionDuration().Get()

	// Assert
	if !ok || duration != 3*time.Hour {
		t.Fatalf("expected 3h duration, got %v (present=%v)", duration, ok)
	}
}

func TestResolutionDurationAbsentWhenUnresolved(t *testing.T) {
	t.Parallel()
	// Arrange
	task := baseTask()
	task.ResolvedAt = domain.None[time.Time]()

	// Act & Assert
	if task.ResolutionDuration().IsPresent() {
		t.Fatal("expected no duration for unresolved task")
	}
}

func TestResolutionDurationAbsentWhenResolvedBeforeCreated(t *testing.T) {
	t.Parallel()
	// Arrange — cloned/imported anomaly: resolved before created.
	task := baseTask()
	task.ResolvedAt = domain.Some(task.CreatedAt.Add(-2 * time.Hour))

	// Act & Assert
	if task.ResolutionDuration().IsPresent() {
		t.Fatal("expected no duration when resolved before created")
	}
}

func TestHasRootCauseWhenOneIsFilled(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if !baseTask().HasRootCause() {
		t.Fatal("expected root cause present")
	}
}

func TestHasRootCauseIsFalseWhenEmpty(t *testing.T) {
	t.Parallel()
	// Arrange
	task := baseTask()
	task.RootCause = ""

	// Act & Assert
	if task.HasRootCause() {
		t.Fatal("expected no root cause")
	}
}
