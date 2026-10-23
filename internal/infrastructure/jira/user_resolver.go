package jira

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

const userSearchPath = "/rest/api/3/user/search"

type userDTO struct {
	AccountID string `json:"accountId"`
}

// Resolve turns a friendly assignee term (name, email, accountId) into a Jira
// accountId. Terms that already look like an accountId are returned as-is.
func (c *Client) Resolve(ctx context.Context, term string) (string, error) {
	if looksLikeAccountID(term) {
		return term, nil
	}

	params := url.Values{}
	params.Set("query", term)
	params.Set("maxResults", "1")
	endpoint, err := c.endpoint(userSearchPath, params)
	if err != nil {
		return "", err
	}

	users, err := getJSON[[]userDTO](ctx, c, endpoint)
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", fmt.Errorf("no Jira user matches %q", term)
	}
	return users[0].AccountID, nil
}

// looksLikeAccountID matches Jira Cloud accountIds: the GDPR form
// "557058:uuid" or the legacy 24-char hex form.
func looksLikeAccountID(term string) bool {
	if strings.Contains(term, ":") {
		return true
	}
	if len(term) != 24 {
		return false
	}
	for _, r := range term {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !isHex {
			return false
		}
	}
	return true
}
