package config

import "github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"

// anthropicConfig holds the resolved Anthropic settings. The AI narrative is a
// pure add-on, so it is enabled only when a key is present.
type anthropicConfig struct {
	apiKey string
	model  string
}

// anthropicConfigFromEnv maps ANTHROPIC_* variables, like a Laravel
// config/services.php entry.
func anthropicConfigFromEnv() anthropicConfig {
	return anthropicConfig{
		apiKey: env("ANTHROPIC_API_KEY"),
		model:  envDefault("ANTHROPIC_MODEL", llm.DefaultModel),
	}
}

func (c anthropicConfig) enabled() bool {
	return c.apiKey != ""
}
