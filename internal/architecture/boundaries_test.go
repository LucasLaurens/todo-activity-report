// Package architecture holds no code: it encodes the hexagonal boundaries as a
// test, so a stray import fails the build instead of a code review.
package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/LucasLaurens/todo-activity-report"

// forbiddenImports lists, per inner layer, the import prefixes it must never
// reach for. The domain knows nothing; the application knows the domain only.
// Swapping Jira for another tracker must therefore leave both untouched.
var forbiddenImports = map[string][]string{
	"../domain": {
		modulePath + "/internal/application",
		modulePath + "/internal/infrastructure",
		modulePath + "/internal/config",
		modulePath + "/internal/i18n",
	},
	"../application": {
		modulePath + "/internal/infrastructure",
		modulePath + "/internal/config",
	},
}

func TestInnerLayersNeverImportOuterOnes(t *testing.T) {
	t.Parallel()

	for layer, forbidden := range forbiddenImports {
		for _, file := range goFilesUnder(t, layer) {
			for _, imported := range importsOf(t, file) {
				for _, prefix := range forbidden {
					if strings.HasPrefix(imported, prefix) {
						t.Errorf("%s imports %s — %s must not depend on it", file, imported, layer)
					}
				}
			}
		}
	}
}

// goFilesUnder lists every .go file of a layer, tests included: a test that
// reaches into infrastructure couples the layer just as much as production code.
func goFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(files) == 0 {
		t.Fatalf("no Go file found under %s — the layout moved, update this test", root)
	}
	return files
}

func importsOf(t *testing.T, path string) []string {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	paths := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		unquoted, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("import path in %s: %v", path, err)
		}
		paths = append(paths, unquoted)
	}
	return paths
}
