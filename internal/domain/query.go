package domain

// Query describes which tasks to retrieve from a provider.
type Query struct {
	Project   string
	Assignees []string
	Period    DateRange
}

func (q Query) HasAssigneeFilter() bool {
	return len(q.Assignees) > 0
}
