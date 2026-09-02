package config

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

const claudeCLIBinary = "claude"

const httpTimeout = 30 * time.Second

// Config is the resolved application configuration. It is built once from the
// environment (see Load) and then hands out wired adapters — the composition
// root, in the spirit of a Laravel config repository. It is the only package
// allowed to name concrete adapters.
type Config struct {
	tracker   TrackerName
	project   string
	anthropic anthropicConfig
	locale    i18n.Locale
}

func Load() (Config, error) {
	loadDotEnv()

	tracker := TrackerName(envDefault("TRACKER", string(defaultTrackerName)))
	if _, registered := trackerBuilders[tracker]; !registered {
		return Config{}, fmt.Errorf("%w %q (known: %s)", ErrUnknownTracker, tracker, strings.Join(knownTrackers(), ", "))
	}

	return Config{
		tracker:   tracker,
		project:   env("TRACKER_PROJECT"),
		anthropic: anthropicConfigFromEnv(),
		locale:    i18n.ParseLocale(env("APP_LOCALE")),
	}, nil
}

// Project is the default project/board key, used when --project is omitted.
func (c Config) Project() string {
	return c.project
}

// NewTranslator builds the translator for the configured locale (default fr).
func (c Config) NewTranslator() i18n.Translator {
	return i18n.NewTranslator(c.locale)
}

// NewUserResolver returns the tracker itself as resolver, or a passthrough when
// resolution is disabled (the raw term then goes to the tracker's query).
func (c Config) NewUserResolver(tracker ports.Tracker, disabled bool) ports.UserResolver {
	if disabled {
		return application.PassthroughResolver{}
	}
	return tracker
}

// NewSummarizer picks the narrative strategy, in order of preference:
//  1. the local `claude` CLI (reuses the Claude Code login, no API key);
//  2. the Anthropic API when ANTHROPIC_API_KEY is set;
//  3. a no-op (metrics only).
//
// The caller never receives nil.
func (c Config) NewSummarizer(aiDisabled bool) ports.Summarizer {
	if aiDisabled {
		return llm.NewNoopSummarizer()
	}
	translator := c.NewTranslator()
	if path, err := exec.LookPath(claudeCLIBinary); err == nil {
		return llm.NewClaudeCLISummarizer(path, translator)
	}
	if c.anthropic.enabled() {
		return llm.NewClaudeSummarizer(c.anthropic.apiKey, c.anthropic.model, translator, &http.Client{Timeout: httpTimeout})
	}
	return llm.NewNoopSummarizer()
}
