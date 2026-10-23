package ports

// ProgressReporter receives the current processing stage so a UI (e.g. a
// terminal spinner) can show where the work stands. The no-op implementation
// keeps the use case free of nil checks.
type ProgressReporter interface {
	Stage(label string)
}

// The stages the use case reports, in order. Each label doubles as the i18n
// message key the spinner translates, so a new stage needs an entry in every
// catalog (the i18n tests check it).
const (
	StageResolve = "stage_resolve"
	StageFetch   = "stage_fetch"
	StageMetrics = "stage_metrics"
	StageSummary = "stage_summary"
)
