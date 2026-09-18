package domain

import "time"

// AssigneeMetrics aggregates a single person's contribution over the period.
type AssigneeMetrics struct {
	Assignee  string
	Total     int
	Resolved  int
	TimeSpent TimeSpent
}

// LabelCount is a generic (label, occurrences) pair used for distributions.
type LabelCount struct {
	Label string
	Count int
}

// TaskTime is one ticket's slice of the period: the time logged on it and the
// share of the total that represents — the answer to "where did the day go?".
type TaskTime struct {
	Key       string
	Summary   string
	Status    string
	TimeSpent TimeSpent
	Share     float64
}

// Metrics is the deterministic aggregation computed from the tasks.
type Metrics struct {
	TotalTasks     int
	ResolvedTasks  int
	TotalTimeSpent TimeSpent
	AvgTimeSpent   TimeSpent
	MTTR           Optional[time.Duration]
	ByAssignee     []AssigneeMetrics
	TimeBreakdown  []TaskTime
	UntimedTasks   int
	ByStatus       []LabelCount
	ByType         []LabelCount
	ByPriority     []LabelCount
	TopRootCauses  []LabelCount
}

func (m Metrics) ResolutionRate() float64 {
	if m.TotalTasks == 0 {
		return 0
	}
	return float64(m.ResolvedTasks) / float64(m.TotalTasks)
}

// Report is the full output: the period, the tasks, their metrics and an
// optional AI narrative.
type Report struct {
	Project   string
	Period    DateRange
	Assignees []string
	Tasks     []Task
	Metrics   Metrics
	Narrative Optional[string]
}
