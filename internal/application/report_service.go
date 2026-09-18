package application

import (
	"context"
	"fmt"
	"sort"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

// ReportService is the central use case: fetch tasks, compute metrics, and
// optionally enrich them with an AI narrative. It depends only on ports
// (dependency inversion), never on concrete adapters.
type ReportService struct {
	provider   ports.TaskProvider
	resolver   ports.UserResolver
	summarizer ports.Summarizer
	progress   ports.ProgressReporter
	calculator MetricsCalculator
}

func NewReportService(provider ports.TaskProvider, resolver ports.UserResolver, summarizer ports.Summarizer, progress ports.ProgressReporter) ReportService {
	return ReportService{
		provider:   provider,
		resolver:   resolver,
		summarizer: summarizer,
		progress:   progress,
		calculator: NewMetricsCalculator(),
	}
}

func (s ReportService) Generate(ctx context.Context, query domain.Query) (domain.Report, error) {
	displayAssignees := query.Assignees

	s.progress.Stage("stage_resolve")
	resolved, err := s.resolveAssignees(ctx, query.Assignees)
	if err != nil {
		return domain.Report{}, fmt.Errorf("resolve assignees: %w", err)
	}
	query.Assignees = resolved

	s.progress.Stage("stage_fetch")
	tasks, err := s.provider.FetchTasks(ctx, query)
	if err != nil {
		return domain.Report{}, fmt.Errorf("fetch tasks: %w", err)
	}

	sortByAttention(tasks)

	s.progress.Stage("stage_metrics")
	report := domain.Report{
		Project:   query.Project,
		Period:    query.Period,
		Assignees: displayAssignees,
		Tasks:     tasks,
		Metrics:   s.calculator.Calculate(tasks, query.Period),
		Narrative: domain.None[string](),
	}

	s.progress.Stage("stage_summary")
	narrative, err := s.summarizer.Summarize(ctx, report)
	if err != nil {
		return domain.Report{}, fmt.Errorf("summarize report: %w", err)
	}
	report.Narrative = narrative

	return report, nil
}

// resolveAssignees maps each friendly assignee term to the tracker identifier,
// preserving order. An empty list (all users) resolves to an empty list.
// sortByAttention puts what blocks first, then what is moving, what shipped,
// what is not started and last what was abandoned. The ticket key breaks ties
// so the report — and the prompt built from it — stays reproducible.
// O(n log n) time, O(1) extra space.
func sortByAttention(tasks []domain.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		left, right := tasks[i].StatusKind().AttentionRank(), tasks[j].StatusKind().AttentionRank()
		if left != right {
			return left < right
		}
		return tasks[i].Key < tasks[j].Key
	})
}

func (s ReportService) resolveAssignees(ctx context.Context, terms []string) ([]string, error) {
	resolved := make([]string, 0, len(terms))
	for _, term := range terms {
		accountID, err := s.resolver.Resolve(ctx, term)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, accountID)
	}
	return resolved, nil
}
