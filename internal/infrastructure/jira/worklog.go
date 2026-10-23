package jira

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

const worklogPathFormat = "/rest/api/3/issue/%s/worklog"

type worklogResponse struct {
	Total    int            `json:"total"`
	Worklogs []worklogEntry `json:"worklogs"`
}

type worklogEntry struct {
	Started          string        `json:"started"`
	TimeSpentSeconds int           `json:"timeSpentSeconds"`
	Author           worklogAuthor `json:"author"`
}

type worklogAuthor struct {
	AccountID string `json:"accountId"`
}

// periodTimeSpent sums the seconds actually logged on an issue within the
// period. When authors is non-empty, only those accountIds' worklogs count, so
// "--user" yields that person's time on that day rather than the team's.
func (c *Client) periodTimeSpent(ctx context.Context, key string, period domain.DateRange, authors map[string]bool) (int, error) {
	total := 0
	startAt := 0
	for {
		page, err := c.fetchWorklogPage(ctx, key, startAt)
		if err != nil {
			return 0, err
		}
		seconds, err := sumWorklogsInPeriod(page.Worklogs, period, authors)
		if err != nil {
			return 0, err
		}
		total += seconds
		startAt += len(page.Worklogs)
		if len(page.Worklogs) == 0 || startAt >= page.Total {
			break
		}
	}
	return total, nil
}

// sumWorklogsInPeriod totals the entries logged within the period by the given
// authors (every author when the set is empty). O(n) time, O(1) space.
func sumWorklogsInPeriod(entries []worklogEntry, period domain.DateRange, authors map[string]bool) (int, error) {
	total := 0
	for _, entry := range entries {
		if len(authors) > 0 && !authors[entry.Author.AccountID] {
			continue
		}
		started, err := parseTime(entry.Started)
		if err != nil {
			return 0, fmt.Errorf("worklog started: %w", err)
		}
		if period.Contains(started) {
			total += entry.TimeSpentSeconds
		}
	}
	return total, nil
}

func (c *Client) fetchWorklogPage(ctx context.Context, key string, startAt int) (worklogResponse, error) {
	params := url.Values{}
	params.Set("startAt", strconv.Itoa(startAt))
	endpoint, err := c.endpoint(fmt.Sprintf(worklogPathFormat, url.PathEscape(key)), params)
	if err != nil {
		return worklogResponse{}, err
	}
	return getJSON[worklogResponse](ctx, c, endpoint)
}

func authorSet(accountIDs []string) map[string]bool {
	if len(accountIDs) == 0 {
		return nil
	}
	authors := make(map[string]bool, len(accountIDs))
	for _, id := range accountIDs {
		authors[id] = true
	}
	return authors
}
