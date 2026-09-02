package jira

import (
	"errors"
	"net/url"
)

var (
	// ErrMissingConfig is returned when the credentials the adapter needs are
	// absent. The application layer only ever sees the error, never the fields.
	ErrMissingConfig = errors.New("missing Jira configuration: set JIRA_BASE_URL, JIRA_EMAIL and JIRA_API_TOKEN")
	// ErrInsecureBaseURL guards the API token: basic auth over plain HTTP would
	// put it on the wire in cleartext.
	ErrInsecureBaseURL = errors.New("JIRA_BASE_URL must use https:// so the API token is never sent in cleartext")
)

// Config holds everything the client needs to talk to a Jira Cloud instance.
type Config struct {
	BaseURL         string
	Email           string
	APIToken        string
	RootCauseField  string
	CorrectionField string
}

// Validate reports whether this adapter can be built. Each adapter owns its own
// rules, so the composition root does not have to know them.
func (c Config) Validate() error {
	if c.BaseURL == "" || c.Email == "" || c.APIToken == "" {
		return ErrMissingConfig
	}
	if !isHTTPS(c.BaseURL) {
		return ErrInsecureBaseURL
	}
	return nil
}

func isHTTPS(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	return err == nil && parsed.Scheme == "https"
}
