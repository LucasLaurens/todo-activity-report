package config_test

import (
	"context"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/config"
	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func loadedConfig(t *testing.T) config.Config {
	t.Helper()
	setJiraEnv(t, "https://example.atlassian.net")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return cfg
}

func builtTracker(t *testing.T, cfg config.Config) ports.Tracker {
	t.Helper()
	tracker, err := cfg.NewTracker()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return tracker
}

func TestNewTrackerIsNonNil(t *testing.T) {
	// Arrange
	cfg := loadedConfig(t)

	// Act
	tracker := builtTracker(t, cfg)

	// Assert
	if tracker == nil {
		t.Fatal("expected a tracker")
	}
}

func TestNewOfflineSummarizerProducesNoNarrative(t *testing.T) {
	// Arrange
	cfg := loadedConfig(t)

	// Act
	narrative, err := cfg.NewOfflineSummarizer().Summarize(context.Background(), domain.Report{})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.IsPresent() {
		t.Fatal("expected no narrative when AI disabled")
	}
}
