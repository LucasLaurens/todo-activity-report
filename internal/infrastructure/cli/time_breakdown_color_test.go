package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

// A breakdown row carries no status text, so colouring it by keywords would
// read the summary instead: "recette de test" would turn a blocked ticket blue.
func TestRenderTimeBreakdownColorsEachRowByItsOwnStatus(t *testing.T) {
	t.Parallel()
	// Arrange
	var buffer bytes.Buffer
	renderer := Renderer{out: &buffer, t: i18n.NewTranslator(i18n.French), color: true}
	report := domain.Report{
		Metrics: domain.Metrics{
			TimeBreakdown: []domain.TaskTime{{
				Key:       "HT-1",
				Summary:   "Reprise de la recette de test",
				Status:    "Bloqué",
				TimeSpent: domain.NewTimeSpent(3600),
				Share:     1,
			}},
		},
	}

	// Act
	if err := renderer.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	row := buffer.String()
	if !strings.Contains(row, ansiRed+"  HT-1") {
		t.Fatalf("expected the row painted red by its status, got %q", row)
	}
}
