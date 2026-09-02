package application_test

import (
	"testing"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/testsupport"
)

func TestCalculateAggregatesTotalsAndAverage(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithTimeSpentHours(2).Build(),
		testsupport.NewTask().WithKey("HT-2").WithTimeSpentHours(4).Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if metrics.TotalTasks != 2 {
		t.Fatalf("expected 2 tasks, got %d", metrics.TotalTasks)
	}
	if metrics.TotalTimeSpent.Hours() != 6 {
		t.Fatalf("expected 6h total, got %v", metrics.TotalTimeSpent.Hours())
	}
	if metrics.AvgTimeSpent.Hours() != 3 {
		t.Fatalf("expected 3h average, got %v", metrics.AvgTimeSpent.Hours())
	}
}

func TestCalculateCountsResolvedTasks(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").Build(),
		testsupport.NewTask().WithKey("HT-2").Unresolved().Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if metrics.ResolvedTasks != 1 {
		t.Fatalf("expected 1 resolved, got %d", metrics.ResolvedTasks)
	}
	if metrics.ResolutionRate() != 0.5 {
		t.Fatalf("expected resolution rate 0.5, got %v", metrics.ResolutionRate())
	}
}

func TestCalculateComputesMeanTimeToResolution(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").ResolvedAfter(2 * time.Hour).Build(),
		testsupport.NewTask().WithKey("HT-2").ResolvedAfter(6 * time.Hour).Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	mttr, present := metrics.MTTR.Get()
	if !present {
		t.Fatal("expected MTTR to be present")
	}
	if mttr != 4*time.Hour {
		t.Fatalf("expected 4h MTTR, got %v", mttr)
	}
}

func TestCalculateRanksRootCausesByFrequency(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithRootCause("Config drift").Build(),
		testsupport.NewTask().WithKey("HT-2").WithRootCause("Config drift").Build(),
		testsupport.NewTask().WithKey("HT-3").WithRootCause("Race condition").Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if len(metrics.TopRootCauses) != 2 {
		t.Fatalf("expected 2 root causes, got %d", len(metrics.TopRootCauses))
	}
	top := metrics.TopRootCauses[0]
	if top.Label != "Config drift" || top.Count != 2 {
		t.Fatalf("expected \"Config drift\" (2) first, got %q (%d)", top.Label, top.Count)
	}
}

func TestCalculateGroupsByAssigneeOrderedByVolume(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithAssignee("Ada").Build(),
		testsupport.NewTask().WithKey("HT-2").WithAssignee("Grace").Build(),
		testsupport.NewTask().WithKey("HT-3").WithAssignee("Grace").Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if len(metrics.ByAssignee) != 2 {
		t.Fatalf("expected 2 assignees, got %d", len(metrics.ByAssignee))
	}
	if metrics.ByAssignee[0].Assignee != "Grace" || metrics.ByAssignee[0].Total != 2 {
		t.Fatalf("expected Grace (2) first, got %q (%d)", metrics.ByAssignee[0].Assignee, metrics.ByAssignee[0].Total)
	}
}

func TestCalculateDistributesByStatusForProgressView(t *testing.T) {
	t.Parallel()
	// Arrange — tickets processed that day across different states.
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithStatus("En cours").Build(),
		testsupport.NewTask().WithKey("HT-2").WithStatus("En cours").Build(),
		testsupport.NewTask().WithKey("HT-3").WithStatus("Bloqué").Build(),
		testsupport.NewTask().WithKey("HT-4").WithStatus("Terminé").Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if len(metrics.ByStatus) != 3 {
		t.Fatalf("expected 3 statuses, got %d: %v", len(metrics.ByStatus), metrics.ByStatus)
	}
	// Ordering is attention-first (see TestCalculateOrdersStatusBreakdownBlockersFirst),
	// so the blocked bucket leads despite being the smallest.
	counts := map[string]int{}
	for _, entry := range metrics.ByStatus {
		counts[entry.Label] = entry.Count
	}
	if counts["En cours"] != 2 || counts["Bloqué"] != 1 || counts["Terminé"] != 1 {
		t.Fatalf("unexpected counts: %v", metrics.ByStatus)
	}
	if metrics.ByStatus[0].Label != "Bloqué" {
		t.Fatalf("expected the blocked bucket first, got %v", metrics.ByStatus)
	}
}

func TestCalculateDistributesByTypeAndPriority(t *testing.T) {
	t.Parallel()
	// Arrange
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithType("Bug").WithPriority("High").Build(),
		testsupport.NewTask().WithKey("HT-2").WithType("Bug").WithPriority("Low").Build(),
		testsupport.NewTask().WithKey("HT-3").WithType("Story").WithPriority("High").Build(),
	}
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(tasks)

	// Assert
	if metrics.ByType[0].Label != "Bug" || metrics.ByType[0].Count != 2 {
		t.Fatalf("expected Bug (2) first, got %+v", metrics.ByType)
	}
	if metrics.ByPriority[0].Label != "High" || metrics.ByPriority[0].Count != 2 {
		t.Fatalf("expected High (2) first, got %+v", metrics.ByPriority)
	}
}

func TestCalculateOnEmptySetIsZeroValued(t *testing.T) {
	t.Parallel()
	// Arrange
	calculator := application.NewMetricsCalculator()

	// Act
	metrics := calculator.Calculate(nil)

	// Assert
	if metrics.TotalTasks != 0 || metrics.AvgTimeSpent.Seconds() != 0 {
		t.Fatalf("expected zero-valued metrics, got %+v", metrics)
	}
	if metrics.MTTR.IsPresent() {
		t.Fatal("expected MTTR to be absent on empty set")
	}
}

func TestCalculateOrdersStatusBreakdownBlockersFirst(t *testing.T) {
	t.Parallel()
	// Arrange — the biggest bucket is the one that needs the least attention.
	tasks := []domain.Task{
		testsupport.NewTask().WithKey("HT-1").WithStatus("À faire").Build(),
		testsupport.NewTask().WithKey("HT-2").WithStatus("À faire").Build(),
		testsupport.NewTask().WithKey("HT-3").WithStatus("À faire").Build(),
		testsupport.NewTask().WithKey("HT-4").WithStatus("Deployed / Completed").Build(),
		testsupport.NewTask().WithKey("HT-5").WithStatus("Code Review").Build(),
		testsupport.NewTask().WithKey("HT-6").WithStatus("Bloqué").Build(),
		testsupport.NewTask().WithKey("HT-7").WithStatus("Annulé").Build(),
	}

	// Act
	metrics := application.NewMetricsCalculator().Calculate(tasks)

	// Assert
	labels := make([]string, 0, len(metrics.ByStatus))
	for _, entry := range metrics.ByStatus {
		labels = append(labels, entry.Label)
	}
	expected := []string{"Bloqué", "Code Review", "Deployed / Completed", "À faire", "Annulé"}
	for i, label := range expected {
		if labels[i] != label {
			t.Fatalf("expected %v, got %v", expected, labels)
		}
	}
}
