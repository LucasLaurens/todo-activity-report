package config_test

import (
	"context"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
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

func TestNewUserResolverPassthroughWhenDisabled(t *testing.T) {
	// Arrange
	cfg := loadedConfig(t)
	resolver := cfg.NewUserResolver(builtTracker(t, cfg), true)

	// Act — passthrough must not hit the network; returns the term as-is.
	got, err := resolver.Resolve(context.Background(), "Ada Lovelace")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, isPassthrough := resolver.(application.PassthroughResolver); !isPassthrough {
		t.Fatalf("expected a PassthroughResolver, got %T", resolver)
	}
	if got != "Ada Lovelace" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}

func TestNewUserResolverReturnsTrackerWhenEnabled(t *testing.T) {
	// Arrange
	cfg := loadedConfig(t)
	tracker := builtTracker(t, cfg)

	// Act — resolution enabled → the tracker itself is the resolver.
	resolver := cfg.NewUserResolver(tracker, false)

	// Assert
	if resolver != tracker {
		t.Fatalf("expected the tracker as resolver, got %T", resolver)
	}
}

func TestNewSummarizerNoopWhenAIDisabled(t *testing.T) {
	// Arrange
	cfg := loadedConfig(t)

	// Act
	narrative, err := cfg.NewSummarizer(true).Summarize(context.Background(), domain.Report{})

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if narrative.IsPresent() {
		t.Fatal("expected no narrative when AI disabled")
	}
}
