package config

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/application"
	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

const httpTimeout = 30 * time.Second

// Config is the resolved application configuration. It is built once from the
// environment (see Load) and then hands out wired adapters — the composition
// root, in the spirit of a Laravel config repository. It is the only package
// allowed to name concrete adapters.
type Config struct {
	tracker     TrackerName
	project     string
	defaultUser string
	anthropic   anthropicConfig
	locale      i18n.Locale
}

func Load() (Config, error) {
	loadDotEnv()

	tracker := TrackerName(envDefault("TRACKER", string(defaultTrackerName)))
	if _, registered := trackerBuilders[tracker]; !registered {
		return Config{}, fmt.Errorf("%w %q (known: %s)", ErrUnknownTracker, tracker, strings.Join(knownTrackers(), ", "))
	}

	return Config{
		tracker:     tracker,
		project:     env("TRACKER_PROJECT"),
		defaultUser: envDefault("TRACKER_USER", env("JIRA_EMAIL")),
		anthropic:   anthropicConfigFromEnv(),
		locale:      i18n.ParseLocale(env("APP_LOCALE")),
	}, nil
}

// Project is the default project/board key, used when --project is omitted.
func (c Config) Project() string {
	return c.project
}

// DefaultAssignee is who the report is about when --user is omitted: the account
// the tool authenticates with. TRACKER_USER names it for any tracker, and the
// Jira adapter's own email is the fallback so a Jira setup needs no new
// variable. Empty means "no default": the report then covers the whole board.
func (c Config) DefaultAssignee() string {
	return c.defaultUser
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

// NewSummarizer wires the narrative strategy behind a fail-safe: the report is
// worth printing even when the model is not reachable, so a failure warns on
// stderr instead of killing the run. The caller never receives nil.
func (c Config) NewSummarizer(aiDisabled bool) ports.Summarizer {
	return llm.NewFailSafeSummarizer(c.narrativeStrategy(aiDisabled), os.Stderr)
}

// narrativeStrategy picks the source of the narrative, in order of preference:
//  1. the local `claude` CLI (reuses the Claude Code login, no API key);
//  2. the Anthropic API when ANTHROPIC_API_KEY is set;
//  3. a no-op (metrics only).
func (c Config) narrativeStrategy(aiDisabled bool) ports.Summarizer {
	if aiDisabled {
		return llm.NewNoopSummarizer()
	}
	translator := c.NewTranslator()
	if path, err := exec.LookPath(llm.DefaultCLIBinary); err == nil {
		return llm.NewClaudeCLISummarizer(path, translator)
	}
	if c.anthropic.enabled() {
		return llm.NewClaudeSummarizer(c.anthropic.apiKey, c.anthropic.model, translator, &http.Client{Timeout: httpTimeout})
	}
	return llm.NewNoopSummarizer()
}
