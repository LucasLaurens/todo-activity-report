package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

const (
	searchPath  = "/rest/api/3/search/jql"
	pageSize    = 100
	fieldsParam = "fields"
)

// Client is a Jira adapter implementing ports.TaskProvider and
// ports.UserResolver.
type Client struct {
	config     Config
	httpClient *http.Client
	mapping    fieldMapping
	fields     string
}

func NewClient(config Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		config:     config,
		httpClient: httpClient,
		mapping: fieldMapping{
			rootCauseField:  config.RootCauseField,
			correctionField: config.CorrectionField,
		},
		fields: buildRequestedFields(config),
	}
}

func (c *Client) FetchTasks(ctx context.Context, query domain.Query) ([]domain.Task, error) {
	jql := buildJQL(query)
	tasks := make([]domain.Task, 0)
	pageToken := ""

	for {
		page, err := c.fetchPage(ctx, jql, pageToken)
		if err != nil {
			return nil, err
		}
		for _, dto := range page.Issues {
			task, err := toTask(dto, c.mapping)
			if err != nil {
				return nil, fmt.Errorf("map issue %s: %w", dto.Key, err)
			}
			tasks = append(tasks, task)
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	// Replace the cumulative timespent with the time actually logged within the
	// period (by the targeted users). One worklog call per ticket — run with
	// bounded concurrency to stay fast without flooding Jira.
	if err := c.enrichWithPeriodTime(ctx, tasks, query.Period, authorSet(query.Assignees)); err != nil {
		return nil, err
	}
	return tasks, nil
}

const maxWorklogConcurrency = 8

// enrichWithPeriodTime fills each task's TimeSpent with its in-period worklog
// total, fetching with a bounded worker pool. Each goroutine writes a distinct
// slice index (no shared mutation), order is preserved, and the first error
// cancels the rest.
func (c *Client) enrichWithPeriodTime(ctx context.Context, tasks []domain.Task, period domain.DateRange, authors map[string]bool) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	semaphore := make(chan struct{}, maxWorklogConcurrency)
	var wg sync.WaitGroup
	var once sync.Once
	var firstErr error

	for i := range tasks {
		select {
		case <-ctx.Done():
			wg.Wait()
			return firstErr
		case semaphore <- struct{}{}:
		}

		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			seconds, err := c.periodTimeSpent(ctx, tasks[index].Key, period, authors)
			if err != nil {
				once.Do(func() {
					firstErr = fmt.Errorf("worklog %s: %w", tasks[index].Key, err)
					cancel()
				})
				return
			}
			tasks[index].TimeSpent = domain.NewTimeSpent(seconds)
		}(i)
	}

	wg.Wait()
	return firstErr
}

func (c *Client) fetchPage(ctx context.Context, jql, pageToken string) (searchResponse, error) {
	endpoint, err := c.searchURL(jql, pageToken)
	if err != nil {
		return searchResponse{}, err
	}
	body, err := c.get(ctx, endpoint)
	if err != nil {
		return searchResponse{}, err
	}
	var decoded searchResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return searchResponse{}, err
	}
	return decoded, nil
}

// get performs an authenticated GET and returns the body on HTTP 200.
func (c *Client) get(ctx context.Context, endpoint string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", c.authHeader())

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira returned %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (c *Client) searchURL(jql, pageToken string) (string, error) {
	base, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + searchPath

	params := url.Values{}
	params.Set("jql", jql)
	params.Set("maxResults", strconv.Itoa(pageSize))
	params.Set(fieldsParam, c.fields)
	if pageToken != "" {
		params.Set("nextPageToken", pageToken)
	}
	base.RawQuery = params.Encode()

	return base.String(), nil
}

// buildRequestedFields computes the comma-separated field list once, at
// construction, rather than on every paginated request.
func buildRequestedFields(config Config) string {
	fields := []string{
		"summary", "status", "assignee", "issuetype", "priority",
		"created", "resolutiondate", "timespent", "aggregatetimespent", "comment",
	}
	if config.RootCauseField != "" {
		fields = append(fields, config.RootCauseField)
	}
	if config.CorrectionField != "" {
		fields = append(fields, config.CorrectionField)
	}
	return strings.Join(fields, ",")
}

func (c *Client) authHeader() string {
	credentials := c.config.Email + ":" + c.config.APIToken
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	return "Basic " + encoded
}

// buildJQL scopes the query to the project, optional assignees, and the tickets
// worked on in the window: time logged that day (worklogDate) OR a status change
// that day (status CHANGED DURING). This captures progress across every state —
// in progress, blocked, released, done — not just resolved tickets.
//
// query.IncludeTouched adds every other update of the day — comments, field
// edits, reassignments — through the ticket's updated timestamp. It is opt-in
// because `updated` bumps on any change by anyone, including bulk edits: on a
// one-day report it buries the tickets the person actually worked on. Jira has
// no commentedDate field, so a day where someone only commented needs it.
func buildJQL(query domain.Query) string {
	clauses := []string{fmt.Sprintf("project = %s", quote(query.Project))}

	if query.HasAssigneeFilter() {
		quoted := make([]string, 0, len(query.Assignees))
		for _, assignee := range query.Assignees {
			quoted = append(quoted, quote(assignee))
		}
		clauses = append(clauses, fmt.Sprintf("assignee in (%s)", strings.Join(quoted, ", ")))
	}

	from := query.Period.FromString()
	to := query.Period.ToString()
	fromStart := quote(from + " 00:00")
	toEnd := quote(to + " 23:59")

	worked := []string{
		fmt.Sprintf("(worklogDate >= %s AND worklogDate <= %s)", quote(from), quote(to)),
		fmt.Sprintf("(status CHANGED DURING (%s, %s))", fromStart, toEnd),
	}
	if query.IncludeTouched {
		worked = append(worked, fmt.Sprintf("(updated >= %s AND updated <= %s)", fromStart, toEnd))
	}
	clauses = append(clauses, "("+strings.Join(worked, " OR ")+")")

	return strings.Join(clauses, " AND ") + " ORDER BY updated DESC"
}

// jqlEscaper escapes the backslash FIRST: escaping only the quote would let a
// term ending with a backslash (`Ada\`) escape the closing quote and inject JQL.
var jqlEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func quote(value string) string {
	return `"` + jqlEscaper.Replace(value) + `"`
}
