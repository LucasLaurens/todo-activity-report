package i18n

import (
	"slices"
	"sort"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/application/ports"
)

var supportedLocales = []Locale{French, English}

func loadCatalogForTest(t *testing.T, locale Locale) map[string]string {
	t.Helper()
	catalog, err := loadCatalog(locale)
	if err != nil {
		t.Fatalf("catalog %q does not load: %v", locale, err)
	}
	return catalog
}

// listSortedKeys is O(k log k) for k keys.
func listSortedKeys(catalog map[string]string) []string {
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestEveryCatalogParses(t *testing.T) {
	t.Parallel()
	for _, locale := range supportedLocales {
		// Act
		catalog := loadCatalogForTest(t, locale)

		// Assert
		if len(catalog) == 0 {
			t.Fatalf("catalog %q is empty", locale)
		}
	}
}

func TestCatalogsShareTheSameKeys(t *testing.T) {
	t.Parallel()
	// Arrange
	reference := listSortedKeys(loadCatalogForTest(t, DefaultLocale))

	for _, locale := range supportedLocales {
		// Act
		keys := listSortedKeys(loadCatalogForTest(t, locale))

		// Assert
		if !slices.Equal(keys, reference) {
			t.Fatalf("catalog %q keys %v differ from %q keys %v", locale, keys, DefaultLocale, reference)
		}
	}
}

func TestEveryProgressStageIsTranslated(t *testing.T) {
	t.Parallel()
	// Arrange
	stages := []string{ports.StageResolve, ports.StageFetch, ports.StageMetrics, ports.StageSummary}

	for _, locale := range supportedLocales {
		// Act
		catalog := loadCatalogForTest(t, locale)

		// Assert
		for _, stage := range stages {
			if _, ok := catalog[stage]; !ok {
				t.Fatalf("catalog %q lacks the progress stage %q", locale, stage)
			}
		}
	}
}
