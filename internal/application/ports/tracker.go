package ports

// Tracker is a work tracker seen from the application: it lists the tasks of a
// period and resolves a friendly assignee term into the identifier its own
// query language expects. Jira, Linear, GitHub Projects or a CSV export are all
// candidates — the use case only ever sees this interface.
type Tracker interface {
	TaskProvider
	UserResolver
}
