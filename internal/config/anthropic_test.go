package config

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

func TestAnthropicConfigDefaultsModelAndDisabledWithoutKey(t *testing.T) {
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

func TestEnvDefaultFallsBack(t *testing.T) {
	t.Setenv("SOME_UNSET_VAR", "")

	// Act & Assert
	if got := envDefault("SOME_UNSET_VAR", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	t.Setenv("SOME_UNSET_VAR", "real")
	if got := envDefault("SOME_UNSET_VAR", "fallback"); got != "real" {
		t.Fatalf("expected real value, got %q", got)
	}
}
