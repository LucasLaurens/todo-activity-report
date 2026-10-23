package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

const (
	messagesEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	// DefaultModel is the model the narrative uses unless ANTHROPIC_MODEL says
	// otherwise. Declared here, next to the API client that sends it.
	DefaultModel     = "claude-sonnet-4-6"
	defaultMaxTokens = 1024
	// maxResponseBytes bounds the answer: 1024 tokens of text weigh a few
	// kilobytes, a megabyte leaves room for the JSON around them.
	maxResponseBytes = 1 << 20
)

// ClaudeSummarizer asks the Anthropic Messages API for a narrative over the
// computed report. It is one Strategy behind the Summarizer port.
type ClaudeSummarizer struct {
	apiKey     string
	model      string
	endpoint   string
	translator i18n.Translator
	httpClient *http.Client
}

func NewClaudeSummarizer(apiKey, model string, translator i18n.Translator, httpClient *http.Client) ClaudeSummarizer {
	if model == "" {
		model = DefaultModel
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return ClaudeSummarizer{apiKey: apiKey, model: model, endpoint: messagesEndpoint, translator: translator, httpClient: httpClient}
}

type messagesRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Messages  []messagePayload `json:"messages"`
}

type messagePayload struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s ClaudeSummarizer) Summarize(ctx context.Context, report domain.Report) (domain.Optional[string], error) {
	raw, err := s.postMessage(ctx, buildPrompt(report, s.translator))
	if err != nil {
		return domain.None[string](), fmt.Errorf("anthropic: %w", err)
	}
	answer, err := decodeAnswer(raw)
	if err != nil {
		return domain.None[string](), fmt.Errorf("anthropic: %w", err)
	}
	return toNarrative(answer), nil
}

// postMessage sends the prompt to the Messages API and returns the raw body of
// a successful answer.
func (s ClaudeSummarizer) postMessage(ctx context.Context, prompt string) ([]byte, error) {
	body, err := json.Marshal(messagesRequest{
		Model:     s.model,
		MaxTokens: defaultMaxTokens,
		Messages:  []messagePayload{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", s.apiKey)
	request.Header.Set("anthropic-version", anthropicVersion)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	raw, err := sanitize.ReadAtMost(response.Body, maxResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		// The fail-safe prints this error on the terminal.
		return nil, fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(sanitize.Printable(string(raw))))
	}
	return raw, nil
}

// decodeAnswer joins the text blocks of a Messages API response.
func decodeAnswer(raw []byte) (string, error) {
	var decoded messagesResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	var answer strings.Builder
	for _, block := range decoded.Content {
		if block.Type == "text" {
			answer.WriteString(block.Text)
		}
	}
	return answer.String(), nil
}
