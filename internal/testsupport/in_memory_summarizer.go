package testsupport

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// InMemorySummarizer is a concrete Summarizer returning a canned narrative and
// recording the report it received.
type InMemorySummarizer struct {
	narrative  domain.Optional[string]
	LastReport domain.Report
	WasCalled  bool
}

func NewInMemorySummarizer(narrative string) *InMemorySummarizer {
	return &InMemorySummarizer{narrative: domain.Some(narrative)}
}

func NewSilentSummarizer() *InMemorySummarizer {
	return &InMemorySummarizer{narrative: domain.None[string]()}
}

func (s *InMemorySummarizer) Summarize(_ context.Context, report domain.Report) (domain.Optional[string], error) {
	s.WasCalled = true
	s.LastReport = report
	return s.narrative, nil
}
