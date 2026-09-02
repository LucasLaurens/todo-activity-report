package application

import "context"

// PassthroughResolver hands the assignee term back unchanged. Null object used
// when resolution is disabled, so the tracker query receives the raw term and
// the use case never deals with a nil resolver.
type PassthroughResolver struct{}

func (PassthroughResolver) Resolve(_ context.Context, term string) (string, error) {
	return term, nil
}
