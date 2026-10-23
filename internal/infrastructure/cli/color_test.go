package cli

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

func TestStatusColorFollowsTheDomainClassification(t *testing.T) {
	t.Parallel()
	// Arrange — one representative status per kind (the kinds themselves are
	// pinned in domain/status_test.go).
	cases := map[string]string{
		"Bloqué":                   ansiRed,
		"En attente retour client": ansiRed,
		"Deployed / Completed":     ansiGreen,
		"Waiting for release":      ansiGreen,
		"In progress dev":          ansiBlue,
		"Waiting for feat env":     ansiBlue,
		"Annulé":                   ansiGrey,
		"À faire":                  "",
	}

	for status, expected := range cases {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			// Act
			color := statusColor(status)

			// Assert
			if color != expected {
				t.Errorf("statusColor(%q) = %q, want %q", status, color, expected)
			}
		})
	}
}

func TestPaintStatusStaysPlainWithoutColor(t *testing.T) {
	t.Parallel()
	// Arrange
	renderer := Renderer{color: false}

	// Act
	painted := renderer.paintByStatus("Bloqué")

	// Assert
	if painted != "Bloqué" {
		t.Fatalf("expected plain status, got %q", painted)
	}
}

func TestRenderMarksAndRedsATicketBlockedInItsComments(t *testing.T) {
	t.Parallel()
	// Arrange
	var buffer bytes.Buffer
	renderer := Renderer{out: &buffer, t: i18n.NewTranslator(i18n.French), color: true}
	report := domain.Report{
		Tasks: []domain.Task{{
			Key:      "HT-2",
			Status:   "In progress dev",
			Summary:  "Extraction OCR",
			Comments: []domain.Comment{{Author: "Ada", Body: "Bloqué par l'environnement."}},
		}},
		Narrative: domain.None[string](),
	}

	// Act
	if err := renderer.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	output := buffer.String()
	marker := i18n.NewTranslator(i18n.French).T("blocking_comment")
	if !strings.Contains(output, ansiRed+"  • HT-2  ·  In progress dev  ·  0m  ·  "+marker+ansiReset) {
		t.Fatalf("expected the ticket to read red and carry the marker, got:\n%q", output)
	}
	if !strings.Contains(output, ansiRed+"    Extraction OCR"+ansiReset) {
		t.Fatalf("expected the whole block to be red, got:\n%q", output)
	}
}

func TestRenderColorsStatusesWhenColorEnabled(t *testing.T) {
	t.Parallel()
	// Arrange
	var buffer bytes.Buffer
	renderer := Renderer{out: &buffer, t: i18n.NewTranslator(i18n.French), color: true}
	report := domain.Report{
		Metrics: domain.Metrics{ByStatus: []domain.LabelCount{{Label: "Bloqué", Count: 2}, {Label: "À faire", Count: 1}}},
		Tasks: []domain.Task{
			{Key: "HT-1", Status: "En cours", Summary: "Ajout du filtre", RootCause: "Token expiré"},
		},
		Narrative: domain.Some("Vue d'ensemble : 1 ticket en cours, 1 bloqué.\nHT-1 (En cours) : filtre en revue.\nHT-9 (Bloqué) : attente client."),
	}

	// Act
	if err := renderer.Render(report); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	output := buffer.String()
	for _, fragment := range []string{
		ansiRed + "  Bloqué   2" + ansiReset,
		ansiBlue + "  • HT-1  ·  En cours  ·  0m" + ansiReset,
		ansiBlue + "    Ajout du filtre" + ansiReset,
		ansiBlue + "    Cause: Token expiré" + ansiReset,
		ansiBlue + "HT-1 (En cours) : filtre en revue." + ansiReset,
		ansiRed + "HT-9 (Bloqué) : attente client." + ansiReset,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%q", fragment, output)
		}
	}
	// The opening narrative line names several statuses: it stays neutral.
	if strings.Contains(output, ansiBlue+"Vue d'ensemble") || strings.Contains(output, ansiRed+"Vue d'ensemble") {
		t.Fatalf("expected the narrative overview line to stay uncolored, got:\n%q", output)
	}
	// Escape codes must not be counted as visible width, or the colored row
	// would be padded differently from the plain one.
	colored := visibleWidth(t, output, "Bloqué")
	plain := visibleWidth(t, output, "À faire")
	if colored != plain {
		t.Fatalf("colored row width %d differs from plain row width %d in:\n%q", colored, plain, output)
	}
}

func visibleWidth(t *testing.T, output, label string) int {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, label) {
			return utf8.RuneCountInString(stripANSI(line))
		}
	}
	t.Fatalf("no line containing %q in:\n%q", label, output)
	return 0
}

func stripANSI(line string) string {
	for _, code := range []string{ansiReset, ansiRed, ansiGreen, ansiBlue} {
		line = strings.ReplaceAll(line, code, "")
	}
	return line
}
