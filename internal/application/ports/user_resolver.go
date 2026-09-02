package ports

import "context"

// UserResolver turns a user-friendly assignee term (name, email, id) into the
// identifier the tracker's own query language expects. PassthroughResolver
// returns the term unchanged, so the use case never deals with a nil dependency.
type UserResolver interface {
	Resolve(ctx context.Context, term string) (string, error)
}
