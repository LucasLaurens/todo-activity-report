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

// trackerBuilder wires one concrete adapter and returns it behind the port. It
// validates its own credentials, so this package holds no tool-specific rule.
type trackerBuilder func() (ports.Tracker, error)

// trackerBuilders is the registry: supporting a new tool means adding one entry
// here plus its adapter package. The domain and the use case stay untouched
// (open/closed).
var trackerBuilders = map[TrackerName]trackerBuilder{
	TrackerJira: buildJiraTracker,
}

// NewTracker builds the configured adapter. Load already refused an unknown
// name, so the only error here is the adapter's own diagnostic (missing
// credentials, insecure URL), which the CLI prints as-is.
func (c Config) NewTracker() (ports.Tracker, error) {
	return trackerBuilders[c.tracker]()
}

// knownTrackers lists the registered names, sorted so the error message is
// reproducible.
func knownTrackers() []string {
	names := make([]string, 0, len(trackerBuilders))
	for name := range trackerBuilders {
		names = append(names, string(name))
	}
	sort.Strings(names)
	return names
}
