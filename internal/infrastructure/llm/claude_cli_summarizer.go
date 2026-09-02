package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

const (
	defaultCLIBinary = "claude"
	cliTimeout       = 120 * time.Second
	cliOutputFormat  = "text"
)

// ClaudeCLISummarizer drives the local `claude` CLI in headless mode, reusing
// the user's existing Claude Code login — no API key required. It is one
// Strategy behind the Summarizer port, alongside ClaudeSummarizer (API key).
type ClaudeCLISummarizer struct {
	binary     string
	timeout    time.Duration
	translator i18n.Translator
}

func NewClaudeCLISummarizer(binary string, translator i18n.Translator) ClaudeCLISummarizer {
	if binary == "" {
		binary = defaultCLIBinary
	}
	return ClaudeCLISummarizer{binary: binary, timeout: cliTimeout, translator: translator}
}

func (s ClaudeCLISummarizer) Summarize(ctx context.Context, report domain.Report) (domain.Optional[string], error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Prompt goes through stdin (not argv): no ARG_MAX limit and no shell
	// interpolation. exec.Command runs the binary directly, never via a shell.
	cmd := exec.CommandContext(ctx, s.binary, "-p", "--output-format", cliOutputFormat)
	cmd.Stdin = strings.NewReader(buildPrompt(report, s.translator))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return domain.None[string](), fmt.Errorf("claude cli: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	narrative := strings.TrimSpace(stdout.String())
	if narrative == "" {
		return domain.None[string](), nil
	}
	return domain.Some(narrative), nil
}
