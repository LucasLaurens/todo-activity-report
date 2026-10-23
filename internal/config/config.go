package config

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/llm"
)

// Config is the resolved application configuration. It is built once from the
// environment (see Load) and then hands out wired adapters — the composition
// root, in the spirit of a Laravel config repository. It names the concrete
// tracker and narrative adapters so the use case only ever sees ports.
type Config struct {
	tracker     trackerEntry
	project     string
	defaultUser string
	anthropic   anthropicConfig
	locale      i18n.Locale
}

func Load() (Config, error) {
	loadDotEnv(os.Stderr)

	name := TrackerName(envDefault("TRACKER", string(defaultTrackerName)))
	tracker, registered := trackerRegistry[name]
	if !registered {
		return Config{}, fmt.Errorf("%w %q (known: %s)", ErrUnknownTracker, name, strings.Join(knownTrackers(), ", "))
	}

	return Config{
		tracker:     tracker,
		project:     env("TRACKER_PROJECT"),
		defaultUser: envDefault("TRACKER_USER", tracker.readDefaultUser()),
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
// selected adapter's own account is the fallback so a setup needs no new
// variable. Empty means "no default": the report then covers the whole board.
func (c Config) DefaultAssignee() string {
	return c.defaultUser
}

// NewTranslator builds the translator for the configured locale (default fr).
func (c Config) NewTranslator() i18n.Translator {
	return i18n.NewTranslator(c.locale)
}

// NewSummarizer wires the narrative strategy behind a fail-safe: the report is
// worth printing even when the model is not reachable, so a failure warns on
// the given writer instead of killing the run. The caller never receives nil.
func (c Config) NewSummarizer(warnings io.Writer) ports.Summarizer {
	return llm.NewFailSafeSummarizer(c.selectNarrativeStrategy(), warnings)
}

// NewOfflineSummarizer is the --no-ai choice: metrics only, nothing leaves the
// machine. A no-op cannot fail, so it needs no fail-safe.
func (c Config) NewOfflineSummarizer() ports.Summarizer {
	return llm.NewNoopSummarizer()
}

// selectNarrativeStrategy picks the source of the narrative, in order of
// preference:
//  1. the local `claude` CLI (reuses the Claude Code login, no API key);
//  2. the Anthropic API when ANTHROPIC_API_KEY is set;
//  3. a no-op (metrics only).
func (c Config) selectNarrativeStrategy() ports.Summarizer {
	translator := c.NewTranslator()
	if path, err := exec.LookPath(llm.DefaultCLIBinary); err == nil {
		return llm.NewClaudeCLISummarizer(path, translator)
	}
	if c.anthropic.enabled() {
		return llm.NewClaudeSummarizer(c.anthropic.apiKey, c.anthropic.model, translator, newHTTPClient())
	}
	return c.NewOfflineSummarizer()
}
