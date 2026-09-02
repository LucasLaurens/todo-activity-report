package ports

// ProgressReporter receives the current processing stage so a UI (e.g. a
// terminal spinner) can show where the work stands. The no-op implementation
// keeps the use case free of nil checks.
type ProgressReporter interface {
	Stage(label string)
}
