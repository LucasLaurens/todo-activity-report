package testsupport

import (
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// TaskBuilder builds domain.Task values for tests with sensible defaults and a
// fluent override API, keeping each test focused on the field it exercises.
type TaskBuilder struct {
	task domain.Task
}

func NewTask() *TaskBuilder {
	created := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)
	return &TaskBuilder{
		task: domain.Task{
			Key:        "HT-1",
			Summary:    "Default summary",
			Assignee:   "Ada Lovelace",
			Type:       "Bug",
			Priority:   "High",
			Status:     "Done",
			RootCause:  "Config drift",
			Correction: "Pinned the version",
			CreatedAt:  created,
			ResolvedAt: domain.Some(created.Add(4 * time.Hour)),
			TimeSpent:  domain.NewTimeSpent(2 * 3600),
		},
	}
}

func (b *TaskBuilder) WithKey(key string) *TaskBuilder {
	b.task.Key = key
	return b
}

func (b *TaskBuilder) WithAssignee(assignee string) *TaskBuilder {
	b.task.Assignee = assignee
	return b
}

func (b *TaskBuilder) WithType(taskType string) *TaskBuilder {
	b.task.Type = taskType
	return b
}

func (b *TaskBuilder) WithStatus(status string) *TaskBuilder {
	b.task.Status = status
	return b
}

func (b *TaskBuilder) WithPriority(priority string) *TaskBuilder {
	b.task.Priority = priority
	return b
}

func (b *TaskBuilder) WithRootCause(rootCause string) *TaskBuilder {
	b.task.RootCause = rootCause
	return b
}

func (b *TaskBuilder) WithComments(comments ...domain.Comment) *TaskBuilder {
	b.task.Comments = comments
	return b
}

func (b *TaskBuilder) WithTimeSpentHours(hours int) *TaskBuilder {
	b.task.TimeSpent = domain.NewTimeSpent(hours * 3600)
	return b
}

func (b *TaskBuilder) ResolvedAfter(d time.Duration) *TaskBuilder {
	b.task.ResolvedAt = domain.Some(b.task.CreatedAt.Add(d))
	return b
}

func (b *TaskBuilder) Unresolved() *TaskBuilder {
	b.task.ResolvedAt = domain.None[time.Time]()
	return b
}

func (b *TaskBuilder) Build() domain.Task {
	return b.task
}
