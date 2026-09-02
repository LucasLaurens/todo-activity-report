package application

import (
	"sort"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

const defaultTopRootCauses = 5

// MetricsCalculator computes the deterministic aggregation from a task set.
// It is a pure, stateless domain service: same input always yields the same
// output, which makes it trivial to unit-test.
type MetricsCalculator struct {
	topRootCauses int
}

func NewMetricsCalculator() MetricsCalculator {
	return MetricsCalculator{topRootCauses: defaultTopRootCauses}
}

func (c MetricsCalculator) Calculate(tasks []domain.Task) domain.Metrics {
	total := totalTimeSpent(tasks)
	return domain.Metrics{
		TotalTasks:     len(tasks),
		ResolvedTasks:  countResolved(tasks),
		TotalTimeSpent: total,
		AvgTimeSpent:   total.DividedBy(len(tasks)),
		MTTR:           meanTimeToResolution(tasks),
		ByAssignee:     groupByAssignee(tasks),
		ByStatus:       statusDistribution(tasks),
		ByType:         distribution(tasks, func(t domain.Task) string { return t.Type }),
		ByPriority:     distribution(tasks, func(t domain.Task) string { return t.Priority }),
		TopRootCauses:  c.topRootCauseDistribution(tasks),
	}
}

func countResolved(tasks []domain.Task) int {
	resolved := 0
	for _, task := range tasks {
		if task.IsResolved() {
			resolved++
		}
	}
	return resolved
}

func totalTimeSpent(tasks []domain.Task) domain.TimeSpent {
	total := domain.NewTimeSpent(0)
	for _, task := range tasks {
		total = total.Add(task.TimeSpent)
	}
	return total
}

func meanTimeToResolution(tasks []domain.Task) domain.Optional[time.Duration] {
	var sum time.Duration
	count := 0
	for _, task := range tasks {
		if duration, ok := task.ResolutionDuration().Get(); ok {
			sum += duration
			count++
		}
	}
	if count == 0 {
		return domain.None[time.Duration]()
	}
	return domain.Some(sum / time.Duration(count))
}

func groupByAssignee(tasks []domain.Task) []domain.AssigneeMetrics {
	index := make(map[string]*domain.AssigneeMetrics)
	for _, task := range tasks {
		entry, ok := index[task.Assignee]
		if !ok {
			entry = &domain.AssigneeMetrics{Assignee: task.Assignee, TimeSpent: domain.NewTimeSpent(0)}
			index[task.Assignee] = entry
		}
		entry.Total++
		entry.TimeSpent = entry.TimeSpent.Add(task.TimeSpent)
		if task.IsResolved() {
			entry.Resolved++
		}
	}

	result := make([]domain.AssigneeMetrics, 0, len(index))
	for _, entry := range index {
		result = append(result, *entry)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Total != result[j].Total {
			return result[i].Total > result[j].Total
		}
		return result[i].Assignee < result[j].Assignee
	})
	return result
}

func distribution(tasks []domain.Task, key func(domain.Task) string) []domain.LabelCount {
	counts := make(map[string]int)
	for _, task := range tasks {
		label := key(task)
		if label == "" {
			continue
		}
		counts[label]++
	}
	return sortedCounts(counts, len(counts))
}

// statusDistribution orders the progress breakdown the way the report reads:
// what blocks first, then what is moving, what shipped, what is not started and
// last what was abandoned — volume then label breaking ties.
// O(n + k log k) time for n tasks and k statuses, O(k) space.
func statusDistribution(tasks []domain.Task) []domain.LabelCount {
	counts := make(map[string]int)
	for _, task := range tasks {
		if task.Status == "" {
			continue
		}
		counts[task.Status]++
	}
	result := make([]domain.LabelCount, 0, len(counts))
	for label, count := range counts {
		result = append(result, domain.LabelCount{Label: label, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		left := domain.ClassifyStatus(result[i].Label).AttentionRank()
		right := domain.ClassifyStatus(result[j].Label).AttentionRank()
		if left != right {
			return left < right
		}
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Label < result[j].Label
	})
	return result
}

func (c MetricsCalculator) topRootCauseDistribution(tasks []domain.Task) []domain.LabelCount {
	counts := make(map[string]int)
	for _, task := range tasks {
		if task.HasRootCause() {
			counts[task.RootCause]++
		}
	}
	return sortedCounts(counts, c.topRootCauses)
}

// sortedCounts turns a count map into a slice ordered by descending count,
// with label as a deterministic tie-breaker, capped at limit entries.
func sortedCounts(counts map[string]int, limit int) []domain.LabelCount {
	result := make([]domain.LabelCount, 0, len(counts))
	for label, count := range counts {
		result = append(result, domain.LabelCount{Label: label, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Label < result[j].Label
	})
	if limit < len(result) {
		result = result[:limit]
	}
	return result
}
