package config_test

import (
	"errors"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/config"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
)

func setJiraEnv(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv("TRACKER", string(config.TrackerJira))
	t.Setenv("JIRA_BASE_URL", baseURL)
	t.Setenv("JIRA_EMAIL", "bot@example.com")
	t.Setenv("JIRA_API_TOKEN", "token")
}

func TestLoadRejectsUnknownTracker(t *testing.T) {
	// Arrange
	t.Setenv("TRACKER", "linear")

	// Act
	_, err := config.Load()

	// Assert
	if !errors.Is(err, config.ErrUnknownTracker) {
		t.Fatalf("expected ErrUnknownTracker, got %v", err)
	}
}

func TestLoadDefaultsToJiraTracker(t *testing.T) {
	// Arrange — TRACKER unset must fall back to the only registered adapter.
	setJiraEnv(t, "https://example.atlassian.net")
	t.Setenv("TRACKER", "")

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := cfg.NewTracker(); err != nil {
		t.Fatalf("expected the default Jira tracker to build, got %v", err)
	}
}

// The credential rules belong to the adapter: the composition root only
// forwards its diagnostic.
func TestNewTrackerForwardsAdapterValidation(t *testing.T) {
	// Arrange
	setJiraEnv(t, "http://example.atlassian.net")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Act
	_, err = cfg.NewTracker()

	// Assert
	if !errors.Is(err, jira.ErrInsecureBaseURL) {
		t.Fatalf("expected jira.ErrInsecureBaseURL, got %v", err)
	}
}

func TestProjectComesFromEnvironment(t *testing.T) {
	// Arrange
	setJiraEnv(t, "https://example.atlassian.net")
	t.Setenv("TRACKER_PROJECT", "ABC")

	// Act
	cfg, err := config.Load()

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project() != "ABC" {
		t.Fatalf("expected project ABC, got %q", cfg.Project())
	}
}
