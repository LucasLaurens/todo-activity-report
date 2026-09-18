package jira

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

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
		for _, entry := range page.Worklogs {
			if len(authors) > 0 && !authors[entry.Author.AccountID] {
				continue
			}
			if !period.Contains(parseTime(entry.Started)) {
				continue
			}
			total += entry.TimeSpentSeconds
		}
		startAt += len(page.Worklogs)
		if len(page.Worklogs) == 0 || startAt >= page.Total {
			break
		}
	}
	return total, nil
}

func (c *Client) fetchWorklogPage(ctx context.Context, key string, startAt int) (worklogResponse, error) {
	endpoint, err := c.worklogURL(key, startAt)
	if err != nil {
		return worklogResponse{}, err
	}
	body, err := c.get(ctx, endpoint)
	if err != nil {
		return worklogResponse{}, err
	}
	var decoded worklogResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return worklogResponse{}, err
	}
	return decoded, nil
}

func (c *Client) worklogURL(key string, startAt int) (string, error) {
	base, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/rest/api/3/issue/" + url.PathEscape(key) + "/worklog"

	params := url.Values{}
	params.Set("startAt", strconv.Itoa(startAt))
	base.RawQuery = params.Encode()

	return base.String(), nil
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
