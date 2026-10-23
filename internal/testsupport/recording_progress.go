package testsupport

// RecordingProgress is a ProgressReporter that records the stages it received,
// in order, so tests can assert progress was reported without any spinner
// output.
type RecordingProgress struct {
	Stages []string
}

func NewRecordingProgress() *RecordingProgress {
	return &RecordingProgress{}
}

func (p *RecordingProgress) Stage(label string) {
	p.Stages = append(p.Stages, label)
}
