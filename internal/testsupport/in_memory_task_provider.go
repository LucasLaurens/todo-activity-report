package testsupport

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// InMemoryTaskProvider is a concrete TaskProvider backed by a slice. It records
// the last query so tests can assert what was requested, without any mocking
// framework.
type InMemoryTaskProvider struct {
	tasks     []domain.Task
	err       error
	LastQuery domain.Query
}

func NewInMemoryTaskProvider(tasks ...domain.Task) *InMemoryTaskProvider {
	return &InMemoryTaskProvider{tasks: tasks}
}

func (p *InMemoryTaskProvider) WithError(err error) *InMemoryTaskProvider {
	p.err = err
	return p
}

func (p *InMemoryTaskProvider) FetchTasks(_ context.Context, query domain.Query) ([]domain.Task, error) {
	p.LastQuery = query
	if p.err != nil {
		return nil, p.err
	}
	return p.tasks, nil
}
