package ports

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// TaskProvider is the inbound data port: any source of executed tasks
// (a tracker API, a file, a DB, test fixtures) implements it.
type TaskProvider interface {
	FetchTasks(ctx context.Context, query domain.Query) ([]domain.Task, error)
}
