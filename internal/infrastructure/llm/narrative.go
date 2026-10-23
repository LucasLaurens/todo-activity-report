package llm

import (
	"strings"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/infrastructure/sanitize"
)

// toNarrative turns a model answer into the report's narrative: escape
// sequences stripped before it reaches the terminal, a blank answer absent.
func toNarrative(answer string) domain.Optional[string] {
	narrative := strings.TrimSpace(sanitize.Printable(answer))
	if narrative == "" {
		return domain.None[string]()
	}
	return domain.Some(narrative)
}
