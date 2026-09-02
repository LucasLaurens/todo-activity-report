package jira

import (
	"errors"
	"testing"
)

func completeConfig(baseURL string) Config {
	return Config{BaseURL: baseURL, Email: "bot@example.com", APIToken: "token"}
}

func TestValidateRejectsNonHTTPSBaseURL(t *testing.T) {
	t.Parallel()
	// Arrange
	config := completeConfig("http://example.atlassian.net")

	// Act
	err := config.Validate()

	// Assert
	if !errors.Is(err, ErrInsecureBaseURL) {
		t.Fatalf("expected ErrInsecureBaseURL, got %v", err)
	}
}

func TestValidateAcceptsHTTPSBaseURL(t *testing.T) {
	t.Parallel()
	// Arrange
	config := completeConfig("https://example.atlassian.net")

	// Act
	err := config.Validate()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsIncompleteConfig(t *testing.T) {
	t.Parallel()
	// Arrange
	cases := map[string]Config{
		"no base url": {Email: "bot@example.com", APIToken: "token"},
		"no email":    {BaseURL: "https://example.atlassian.net", APIToken: "token"},
		"no token":    {BaseURL: "https://example.atlassian.net", Email: "bot@example.com"},
	}

	for name, config := range cases {
		// Act
		err := config.Validate()

		// Assert
		if !errors.Is(err, ErrMissingConfig) {
			t.Fatalf("%s: expected ErrMissingConfig, got %v", name, err)
		}
	}
}
