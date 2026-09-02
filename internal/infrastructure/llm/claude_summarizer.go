package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

const (
	messagesEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	defaultModel     = "claude-sonnet-4-6"
	defaultMaxTokens = 1024
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
		model = defaultModel
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
	payload := messagesRequest{
		Model:     s.model,
		MaxTokens: defaultMaxTokens,
		Messages: []messagePayload{
			{Role: "user", Content: buildPrompt(report, s.translator)},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return domain.None[string](), err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return domain.None[string](), err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", s.apiKey)
	request.Header.Set("anthropic-version", anthropicVersion)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return domain.None[string](), err
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return domain.None[string](), err
	}
	if response.StatusCode != http.StatusOK {
		return domain.None[string](), fmt.Errorf("anthropic returned %d: %s", response.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded messagesResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return domain.None[string](), err
	}

	narrative := joinTextBlocks(decoded.Content)
	if narrative == "" {
		return domain.None[string](), nil
	}
	return domain.Some(narrative), nil
}

func joinTextBlocks(blocks []contentBlock) string {
	var builder strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			builder.WriteString(block.Text)
		}
	}
	return strings.TrimSpace(builder.String())
}
