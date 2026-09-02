package jira

import "encoding/json"

// searchResponse mirrors the subset of Jira's /search/jql payload we consume.
// The endpoint paginates by token: nextPageToken is empty on the last page.
type searchResponse struct {
	Issues        []issueDTO `json:"issues"`
	NextPageToken string     `json:"nextPageToken"`
}

// issueDTO keeps fields raw so the mapper can decode both the known schema and
// dynamic custom fields from the same payload.
type issueDTO struct {
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields"`
}

type issueFields struct {
	Summary            string        `json:"summary"`
	Status             namedValue    `json:"status"`
	Assignee           *assignee     `json:"assignee"`
	IssueType          namedValue    `json:"issuetype"`
	Priority           *namedValue   `json:"priority"`
	Created            string        `json:"created"`
	ResolutionDate     *string       `json:"resolutiondate"`
	TimeSpent          *int          `json:"timespent"`
	AggregateTimeSpent *int          `json:"aggregatetimespent"`
	Comment            *commentField `json:"comment"`
}

// commentField mirrors the issue "comment" field returned inline by the search
// endpoint, sparing a per-ticket round trip. Comments arrive oldest-first.
type commentField struct {
	Comments []commentDTO `json:"comments"`
}

// commentDTO holds one comment. Body is decoded as any: Jira returns an
// Atlassian Document Format object (API v3), occasionally a plain string.
type commentDTO struct {
	Author *assignee `json:"author"`
	Body   any       `json:"body"`
}

type namedValue struct {
	Name string `json:"name"`
}

type assignee struct {
	DisplayName string `json:"displayName"`
}
