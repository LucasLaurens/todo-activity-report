package testsupport

import (
	"context"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// InMemorySummarizer is a concrete Summarizer returning a canned narrative and
// recording the report it received.
type InMemorySummarizer struct {
	narrative  domain.Optional[string]
	err        error
	LastReport domain.Report
	WasCalled  bool
}

func NewInMemorySummarizer(narrative string) *InMemorySummarizer {
	return &InMemorySummarizer{narrative: domain.Some(narrative)}
}

func NewSilentSummarizer() *InMemorySummarizer {
	return &InMemorySummarizer{narrative: domain.None[string]()}
}

// WithError turns the summarizer into a broken one: it records the call, then
// fails like a model that timed out or an unreachable API.
func (s *InMemorySummarizer) WithError(err error) *InMemorySummarizer {
	s.err = err
	return s
}

func (s *InMemorySummarizer) Summarize(_ context.Context, report domain.Report) (domain.Optional[string], error) {
	s.WasCalled = true
	s.LastReport = report
	if s.err != nil {
		return domain.None[string](), s.err
	}
	return s.narrative, nil
}
