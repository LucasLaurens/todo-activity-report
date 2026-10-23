package config

import (
	"net/http"
	"testing"
)

func newRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return request
}

// Go copies the Authorization header to a redirect on the same host whatever
// the scheme: following https → http would send the Jira token in clear.
func TestHTTPClientRefusesARedirectFromHTTPSToHTTP(t *testing.T) {
	t.Parallel()
	// Arrange
	client := newHTTPClient()
	via := []*http.Request{newRequest(t, "https://acme.atlassian.net/rest/api/3/search/jql")}

	// Act
	err := client.CheckRedirect(newRequest(t, "http://acme.atlassian.net/rest/api/3/search/jql"), via)

	// Assert
	if err == nil {
		t.Fatal("expected the downgrade to be refused")
	}
}

func TestHTTPClientFollowsARedirectThatStaysOnHTTPS(t *testing.T) {
	t.Parallel()
	// Arrange
	client := newHTTPClient()
	via := []*http.Request{newRequest(t, "https://acme.atlassian.net/a")}

	// Act
	err := client.CheckRedirect(newRequest(t, "https://acme.atlassian.net/b"), via)

	// Assert
	if err != nil {
		t.Fatalf("expected an https redirect to be followed, got %v", err)
	}
}

func TestHTTPClientHasATimeout(t *testing.T) {
	t.Parallel()
	// Act
	client := newHTTPClient()

	// Assert
	if client.Timeout != httpTimeout {
		t.Fatalf("expected timeout %s, got %s", httpTimeout, client.Timeout)
	}
}
