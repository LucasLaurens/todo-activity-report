package testsupport

// NoopProgress is a ProgressReporter that records the stages it received, so
// tests can assert progress was reported without any spinner output.
type NoopProgress struct {
	Stages []string
}

func NewNoopProgress() *NoopProgress {
	return &NoopProgress{}
}

func (p *NoopProgress) Stage(label string) {
	p.Stages = append(p.Stages, label)
}
