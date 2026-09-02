package llm

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// NoopSummarizer is the default Summarizer: it produces no narrative. Using it
// instead of a nil dependency keeps the use case free of nil checks.
type NoopSummarizer struct{}

func NewNoopSummarizer() NoopSummarizer {
	return NoopSummarizer{}
}

func (NoopSummarizer) Summarize(_ context.Context, _ domain.Report) (domain.Optional[string], error) {
	return domain.None[string](), nil
}
