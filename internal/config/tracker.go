package config

import (
	"errors"
	"sort"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
)

// ErrUnknownTracker names a TRACKER value no adapter is registered for.
var ErrUnknownTracker = errors.New("unknown tracker")

// TrackerName is the key selecting which tracker adapter to build.
type TrackerName string

const (
	// TrackerJira is the only adapter shipped today; it is also the default so
	// existing setups keep working without a new variable.
	TrackerJira TrackerName = "jira"

	defaultTrackerName = TrackerJira
)

// trackerEntry is everything this package knows about one tool. build wires
// the adapter, which validates its own credentials; defaultUserEnv names the
// variable holding the account the adapter signs in with, the person a bare
// run reports on when TRACKER_USER is unset. No tool-specific rule leaks into
// Load.
type trackerEntry struct {
	build          func() (ports.Tracker, error)
	defaultUserEnv string
}

// readDefaultUser returns the adapter's own account, empty when it has none.
func (e trackerEntry) readDefaultUser() string {
	if e.defaultUserEnv == "" {
		return ""
	}
	return env(e.defaultUserEnv)
}

// trackerRegistry maps TRACKER to its entry: supporting a new tool means adding
// one entry here plus its adapter package. The domain and the use case stay
// untouched (open/closed).
var trackerRegistry = map[TrackerName]trackerEntry{
	TrackerJira: {build: buildJiraTracker, defaultUserEnv: jiraEmailEnv},
}

// NewTracker builds the configured adapter. Load already refused an unknown
// name, so the only error here is the adapter's own diagnostic (missing
// credentials, insecure URL), which the CLI prints as-is.
func (c Config) NewTracker() (ports.Tracker, error) {
	return c.tracker.build()
}

// knownTrackers lists the registered names, sorted so the error message is
// reproducible.
func knownTrackers() []string {
	names := make([]string, 0, len(trackerRegistry))
	for name := range trackerRegistry {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}
