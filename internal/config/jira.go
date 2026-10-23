package config

import (
	"net/http"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/jira"
)

// jiraEmailEnv is both a Jira credential and the account a bare run reports on.
const jiraEmailEnv = "JIRA_EMAIL"

// buildJiraTracker maps JIRA_* environment variables to the Jira adapter and
// hands it back behind the port. One file = one adapter, like a Laravel
// config/jira.php.
func buildJiraTracker() (ports.Tracker, error) {
	config := jira.Config{
		BaseURL:         env("JIRA_BASE_URL"),
		Email:           env(jiraEmailEnv),
		APIToken:        env("JIRA_API_TOKEN"),
		RootCauseField:  env("JIRA_ROOT_CAUSE_FIELD"),
		CorrectionField: env("JIRA_CORRECTION_FIELD"),
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return jira.NewClient(config, &http.Client{Timeout: httpTimeout}), nil
}
