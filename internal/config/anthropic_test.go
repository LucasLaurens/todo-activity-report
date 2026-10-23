package config

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

func TestAnthropicConfigDefaultsModelAndDisabledWithoutKey(t *testing.T) {
	// Arrange
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_MODEL", "")

	// Act
	cfg := anthropicConfigFromEnv()

	// Assert
	if cfg.model != llm.DefaultModel {
		t.Fatalf("expected default model, got %q", cfg.model)
	}
	if cfg.enabled() {
		t.Fatal("expected disabled without an API key")
	}
}

func TestAnthropicConfigEnabledWithKeyAndCustomModel(t *testing.T) {
	// Arrange
	t.Setenv("ANTHROPIC_API_KEY", "secret")
	t.Setenv("ANTHROPIC_MODEL", "claude-custom")

	// Act
	cfg := anthropicConfigFromEnv()

	// Assert
	if cfg.model != "claude-custom" {
		t.Fatalf("expected custom model, got %q", cfg.model)
	}
	if !cfg.enabled() {
		t.Fatal("expected enabled with an API key")
	}
}

func TestEnvDefaultFallsBackWhenTheVariableIsEmpty(t *testing.T) {
	// Arrange
	t.Setenv("CONFIG_TEST_VAR", "")

	// Act
	got := envDefault("CONFIG_TEST_VAR", "fallback")

	// Assert
	if got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestEnvDefaultPrefersTheVariableWhenSet(t *testing.T) {
	// Arrange
	t.Setenv("CONFIG_TEST_VAR", "real")

	// Act
	got := envDefault("CONFIG_TEST_VAR", "fallback")

	// Assert
	if got != "real" {
		t.Fatalf("expected real value, got %q", got)
	}
}
