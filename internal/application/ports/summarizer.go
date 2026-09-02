package ports

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// Summarizer turns computed metrics into a human narrative. The default
// implementation is a no-op, so callers never deal with a nil dependency.
type Summarizer interface {
	Summarize(ctx context.Context, report domain.Report) (domain.Optional[string], error)
}
