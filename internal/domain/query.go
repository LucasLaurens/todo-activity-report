package domain

// Query describes which tasks to retrieve from a provider.
type Query struct {
	Project   string
	Assignees []string
	Period    DateRange
	// IncludeTouched widens the scope from work actually done (time logged or a
	// status moved) to every ticket updated in the period — a comment, a field
	// edit, a bulk change, whoever made it.
	IncludeTouched bool
}

func (q Query) HasAssigneeFilter() bool {
	return len(q.Assignees) > 0
}
