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

// Calculate aggregates the tasks over the period they were pulled for. The
// period matters: a ticket resolved months ago and merely touched during the
// period is not one of its resolutions.
func (c MetricsCalculator) Calculate(tasks []domain.Task, period domain.DateRange) domain.Metrics {
	total := sumTimeSpent(tasks)
	breakdown, untimed := buildTimeBreakdown(tasks, total)
	return domain.Metrics{
		TotalTasks:     len(tasks),
		ResolvedTasks:  countResolvedWithin(tasks, period),
		TotalTimeSpent: total,
		AvgTimeSpent:   total.DividedBy(len(tasks)),
		MTTR:           computeMeanTimeToResolution(tasks, period),
		ByAssignee:     groupByAssignee(tasks, period),
		TimeBreakdown:  breakdown,
		UntimedTasks:   untimed,
		ByStatus:       computeStatusDistribution(tasks),
		ByType:         computeDistribution(tasks, func(t domain.Task) string { return t.Type }),
		ByPriority:     computeDistribution(tasks, func(t domain.Task) string { return t.Priority }),
		TopRootCauses:  c.computeTopRootCauses(tasks),
	}
}

func countResolvedWithin(tasks []domain.Task, period domain.DateRange) int {
	resolved := 0
	for _, task := range tasks {
		if task.IsResolvedWithin(period) {
			resolved++
		}
	}
	return resolved
}

func sumTimeSpent(tasks []domain.Task) domain.TimeSpent {
	total := domain.NewTimeSpent(0)
	for _, task := range tasks {
		total = total.Add(task.TimeSpent)
	}
	return total
}

// computeMeanTimeToResolution averages creation → resolution over the tickets resolved
// *within* the period: an older resolution says nothing about this period.
func computeMeanTimeToResolution(tasks []domain.Task, period domain.DateRange) domain.Optional[time.Duration] {
	var sum time.Duration
	count := 0
	for _, task := range tasks {
		if !task.IsResolvedWithin(period) {
			continue
		}
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

func groupByAssignee(tasks []domain.Task, period domain.DateRange) []domain.AssigneeMetrics {
	index := make(map[string]*domain.AssigneeMetrics)
	for _, task := range tasks {
		entry, ok := index[task.Assignee]
		if !ok {
			entry = &domain.AssigneeMetrics{Assignee: task.Assignee, TimeSpent: domain.NewTimeSpent(0)}
			index[task.Assignee] = entry
		}
		entry.Total++
		entry.TimeSpent = entry.TimeSpent.Add(task.TimeSpent)
		if task.IsResolvedWithin(period) {
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

// computeDistribution counts the tasks per label, biggest first.
// O(n + k log k) time for n tasks and k labels, O(k) space.
func computeDistribution(tasks []domain.Task, label func(domain.Task) string) []domain.LabelCount {
	counts := countLabels(tasks, label)
	return sortLabelCounts(counts, rankEqually, len(counts))
}

// computeStatusDistribution orders the progress breakdown the way the report
// reads: what blocks first, then what is moving, what shipped, what is not
// started and last what was abandoned — volume then label breaking ties.
// O(n + k log k) time for n tasks and k statuses, O(k) space.
func computeStatusDistribution(tasks []domain.Task) []domain.LabelCount {
	counts := countLabels(tasks, func(t domain.Task) string { return t.Status })
	return sortLabelCounts(counts, rankStatusByAttention, len(counts))
}

func (c MetricsCalculator) computeTopRootCauses(tasks []domain.Task) []domain.LabelCount {
	counts := make(map[string]int)
	for _, task := range tasks {
		if task.HasRootCause() {
			counts[task.RootCause]++
		}
	}
	return sortLabelCounts(counts, rankEqually, c.topRootCauses)
}

// countLabels tallies the tasks per label, skipping the ones without a label.
// O(n) time, O(k) space.
func countLabels(tasks []domain.Task, label func(domain.Task) string) map[string]int {
	counts := make(map[string]int)
	for _, task := range tasks {
		if value := label(task); value != "" {
			counts[value]++
		}
	}
	return counts
}

func rankEqually(string) int {
	return 0
}

func rankStatusByAttention(status string) int {
	return domain.ClassifyStatus(status).AttentionRank()
}

// sortLabelCounts orders the counts by rank, then by descending count, with the
// label as a deterministic tie-breaker, capped at limit entries. Each rank is
// computed once, not on every comparison.
// O(k log k) time, O(k) space for k labels.
func sortLabelCounts(counts map[string]int, rank func(label string) int, limit int) []domain.LabelCount {
	result := make([]domain.LabelCount, 0, len(counts))
	ranks := make(map[string]int, len(counts))
	for label, count := range counts {
		result = append(result, domain.LabelCount{Label: label, Count: count})
		ranks[label] = rank(label)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := result[i], result[j]
		if ranks[left.Label] != ranks[right.Label] {
			return ranks[left.Label] < ranks[right.Label]
		}
		if left.Count != right.Count {
			return left.Count > right.Count
		}
		return left.Label < right.Label
	})
	if limit < len(result) {
		result = result[:limit]
	}
	return result
}
