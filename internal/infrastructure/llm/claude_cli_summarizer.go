package llm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

const (
	// DefaultCLIBinary is the local Claude Code executable looked up on PATH.
	DefaultCLIBinary = "claude"
	// cliTimeout budgets the whole narrative. A day with forty tickets and their
	// comments is a long prompt, and the model answers one line per ticket, so
	// two minutes were not always enough.
	cliTimeout      = 4 * time.Minute
	cliOutputFormat = "text"
	cliWorkDirGlob  = "todo-activity-report-claude-*"
)

// cliIsolationArgs keep the model a plain text generator. Ticket summaries and
// comments are written by anyone with access to the board, so a crafted one
// must not be able to drive tools on this machine: no built-in tool, no MCP
// server, the user's own settings only (a project's hooks would otherwise run
// from the working directory), and no transcript of internal tickets on disk.
var cliIsolationArgs = []string{
	"--tools", "",
	"--strict-mcp-config",
	"--setting-sources", "user",
	"--no-session-persistence",
}

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
		binary = DefaultCLIBinary
	}
	return ClaudeCLISummarizer{binary: binary, timeout: cliTimeout, translator: translator}
}

func (s ClaudeCLISummarizer) Summarize(ctx context.Context, report domain.Report) (domain.Optional[string], error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// An empty working directory: no CLAUDE.md, .claude/ settings or .env of
	// wherever the report was launched from reaches the model.
	workDir, err := os.MkdirTemp("", cliWorkDirGlob)
	if err != nil {
		return domain.None[string](), fmt.Errorf("claude cli work dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Prompt goes through stdin (not argv): no ARG_MAX limit and no shell
	// interpolation. exec.Command runs the binary directly, never via a shell.
	args := append([]string{"-p", "--output-format", cliOutputFormat}, cliIsolationArgs...)
	cmd := exec.CommandContext(ctx, s.binary, args...)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(buildPrompt(report, s.translator))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// A killed process reports "signal: killed" with an empty stderr, which
		// says nothing: name the budget it blew instead.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return domain.None[string](), fmt.Errorf("claude cli timed out after %s (shorten the period or pass --no-ai)", s.timeout)
		}
		return domain.None[string](), fmt.Errorf("claude cli: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	narrative := strings.TrimSpace(sanitize.Printable(stdout.String()))
	if narrative == "" {
		return domain.None[string](), nil
	}
	return domain.Some(narrative), nil
}
