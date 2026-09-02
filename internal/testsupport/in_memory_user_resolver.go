package testsupport

import "context"

// InMemoryUserResolver is a concrete UserResolver: it maps known terms to
// accountIds and passes unknown terms through unchanged.
type InMemoryUserResolver struct {
	mapping map[string]string
}

func NewPassthroughResolver() *InMemoryUserResolver {
	return &InMemoryUserResolver{mapping: map[string]string{}}
}

func NewInMemoryUserResolver(mapping map[string]string) *InMemoryUserResolver {
	return &InMemoryUserResolver{mapping: mapping}
}

func (r *InMemoryUserResolver) Resolve(_ context.Context, term string) (string, error) {
	if accountID, ok := r.mapping[term]; ok {
		return accountID, nil
	}
	return term, nil
}
