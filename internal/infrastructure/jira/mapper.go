package jira

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

const jiraTimeLayout = "2006-01-02T15:04:05.999-0700"

const unassignedLabel = "Unassigned"

// fieldMapping names the optional custom fields holding root cause and fix.
type fieldMapping struct {
	rootCauseField  string
	correctionField string
}

// toTask converts a raw Jira issue into a domain Task. Boundary nil-handling
// lives here so the domain stays free of pointers.
func toTask(dto issueDTO, mapping fieldMapping) (domain.Task, error) {
	var fields issueFields
	if err := json.Unmarshal(dto.Fields, &fields); err != nil {
		return domain.Task{}, err
	}

	custom := make(map[string]any)
	if err := json.Unmarshal(dto.Fields, &custom); err != nil {
		return domain.Task{}, err
	}

	// This root-cause field bundles cause + correction in one template; a
	// dedicated correction field, when configured, takes precedence.
	rootCause, correction := splitRootCause(textField(custom, mapping.rootCauseField))
	if separate := textField(custom, mapping.correctionField); separate != "" {
		correction = separate
	}

	// Every string below is tracker-controlled text about to be printed on a
	// terminal: sanitize on ingress so no escape sequence ever reaches it.
	return domain.Task{
		Key:        sanitize.Printable(dto.Key),
		Summary:    sanitize.Printable(fields.Summary),
		Assignee:   sanitize.Printable(assigneeName(fields.Assignee)),
		Type:       sanitize.Printable(fields.IssueType.Name),
		Priority:   sanitize.Printable(priorityName(fields.Priority)),
		Status:     sanitize.Printable(fields.Status.Name),
		RootCause:  rootCause,
		Correction: correction,
		CreatedAt:  parseTime(fields.Created),
		ResolvedAt: parseOptionalTime(fields.ResolutionDate),
		TimeSpent:  domain.NewTimeSpent(effectiveTimeSpent(fields)),
		Comments:   toComments(fields.Comment),
	}, nil
}

// maxCommentsPerTask caps how many comments travel with a task. Only the most
// recent ones carry the current state of the ticket; keeping the tail bounds
// prompt size (and cost) regardless of how chatty a ticket is.
const maxCommentsPerTask = 5

// toComments keeps the most recent comments (the field arrives oldest-first),
// flattening each ADF body to plain text and dropping empty ones.
func toComments(field *commentField) []domain.Comment {
	if field == nil || len(field.Comments) == 0 {
		return nil
	}
	recent := field.Comments
	if len(recent) > maxCommentsPerTask {
		recent = recent[len(recent)-maxCommentsPerTask:]
	}
	comments := make([]domain.Comment, 0, len(recent))
	for _, entry := range recent {
		body := strings.TrimSpace(sanitize.Printable(flattenValue(entry.Body)))
		if body == "" {
			continue
		}
		comments = append(comments, domain.Comment{Author: sanitize.Printable(assigneeName(entry.Author)), Body: body})
	}
	return comments
}

func assigneeName(value *assignee) string {
	if value == nil || value.DisplayName == "" {
		return unassignedLabel
	}
	return value.DisplayName
}

func priorityName(value *namedValue) string {
	if value == nil {
		return ""
	}
	return value.Name
}

func effectiveTimeSpent(fields issueFields) int {
	if fields.TimeSpent != nil {
		return *fields.TimeSpent
	}
	if fields.AggregateTimeSpent != nil {
		return *fields.AggregateTimeSpent
	}
	return 0
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(jiraTimeLayout, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func parseOptionalTime(value *string) domain.Optional[time.Time] {
	if value == nil || *value == "" {
		return domain.None[time.Time]()
	}
	parsed, err := time.Parse(jiraTimeLayout, *value)
	if err != nil {
		return domain.None[time.Time]()
	}
	return domain.Some(parsed)
}

// textField reads a configured custom field, supporting both plain-text fields
// and Atlassian Document Format bodies.
func textField(fields map[string]any, key string) string {
	if key == "" {
		return ""
	}
	raw, ok := fields[key]
	if !ok || raw == nil {
		return ""
	}
	return sanitize.Printable(flattenValue(raw))
}

// flattenValue renders a decoded JSON value as plain text: a string verbatim, an
// Atlassian Document Format object walked into text, anything else empty.
func flattenValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case map[string]any:
		return extractADFText(value)
	default:
		return ""
	}
}

// adfBlockTypes are ADF nodes that introduce a line break, so the structured
// Problème/Cause/Correction/Prévention template (one section per paragraph or
// bullet) survives flattening and stays parseable by splitRootCause.
var adfBlockTypes = map[string]bool{
	"paragraph":  true,
	"heading":    true,
	"listItem":   true,
	"blockquote": true,
	"codeBlock":  true,
}

// extractADFText walks an Atlassian Document Format tree into plain text,
// preserving line breaks (hard breaks and block boundaries) while concatenating
// inline runs verbatim.
func extractADFText(node map[string]any) string {
	var b strings.Builder
	writeADFText(&b, node)
	return strings.TrimSpace(b.String())
}

func writeADFText(b *strings.Builder, node map[string]any) {
	switch nodeType(node) {
	case "text":
		if text, ok := node["text"].(string); ok {
			b.WriteString(text)
		}
		return
	case "hardBreak":
		b.WriteByte('\n')
		return
	}
	if adfBlockTypes[nodeType(node)] {
		ensureNewline(b)
	}
	if nodeType(node) == "listItem" {
		b.WriteString(listItemMarker)
	}
	content, ok := node["content"].([]any)
	if !ok {
		return
	}
	for _, child := range content {
		if childNode, ok := child.(map[string]any); ok {
			writeADFText(b, childNode)
		}
	}
}

func nodeType(node map[string]any) string {
	value, _ := node["type"].(string)
	return value
}

// listItemMarker prefixes flattened list items; a paragraph nested directly in
// one must not break the line right after it.
const listItemMarker = "- "

func ensureNewline(b *strings.Builder) {
	if b.Len() == 0 {
		return
	}
	current := b.String()
	if strings.HasSuffix(current, "\n") || strings.HasSuffix(current, listItemMarker) {
		return
	}
	b.WriteByte('\n')
}
