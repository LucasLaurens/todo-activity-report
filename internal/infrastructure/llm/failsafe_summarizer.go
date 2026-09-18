package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// FailSafeSummarizer decorates a Summarizer so a failed narrative never costs
// the report: the metrics are already computed and the narrative is a bonus, so
// a timed-out CLI or an unreachable API degrades to "no narrative" plus a
// warning, instead of an exit code and an empty terminal.
type FailSafeSummarizer struct {
	inner    ports.Summarizer
	warnings io.Writer
}

func NewFailSafeSummarizer(inner ports.Summarizer, warnings io.Writer) FailSafeSummarizer {
	return FailSafeSummarizer{inner: inner, warnings: warnings}
}

func (s FailSafeSummarizer) Summarize(ctx context.Context, report domain.Report) (domain.Optional[string], error) {
	narrative, err := s.inner.Summarize(ctx, report)
	if err != nil {
		fmt.Fprintf(s.warnings, "warning: no AI summary (%v)\n", err)
		return domain.None[string](), nil
	}
	return narrative, nil
}
