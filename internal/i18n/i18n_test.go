package i18n_test

import (
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/i18n"
)

func TestParseLocale(t *testing.T) {
	t.Parallel()
	cases := map[string]i18n.Locale{
		"":        i18n.French,
		"fr":      i18n.French,
		"FR":      i18n.French,
		"en":      i18n.English,
		"English": i18n.English,
		"unknown": i18n.French, // falls back to default
	}
	for input, want := range cases {
		if got := i18n.ParseLocale(input); got != want {
			t.Fatalf("ParseLocale(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTranslatorReturnsLocalizedMessage(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if got := i18n.NewTranslator(i18n.French).T("overview"); got != "Synthèse" {
		t.Fatalf("expected French overview, got %q", got)
	}
	if got := i18n.NewTranslator(i18n.English).T("overview"); got != "Overview" {
		t.Fatalf("expected English overview, got %q", got)
	}
}

func TestTranslatorFormatsArguments(t *testing.T) {
	t.Parallel()
	// Act
	got := i18n.NewTranslator(i18n.French).T("tasks_header", 5, 14)

	// Assert
	if got != "Tickets (5 sur 14)" {
		t.Fatalf("unexpected formatted message: %q", got)
	}
}

func TestTranslatorUnknownKeyReturnsKey(t *testing.T) {
	t.Parallel()
	// Act & Assert
	if got := i18n.NewTranslator(i18n.French).T("does_not_exist"); got != "does_not_exist" {
		t.Fatalf("expected the key back, got %q", got)
	}
}

func TestTranslatorLocaleAccessor(t *testing.T) {
	t.Parallel()
	if i18n.NewTranslator(i18n.English).Locale() != i18n.English {
		t.Fatal("expected English locale")
	}
}
