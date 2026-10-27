package importcheck

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestOnlyPackageMainImportsWails keeps Wails out of the packages the CLI and the headless server link: every file that
// imports its application package declares package main.
func TestOnlyPackageMainImportsWails(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	skip := []string{"node_modules", ".git", ".claude", "tmp", "site", "dist", "bin", "frontend", "web"}
	checked := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && slices.Contains(skip, d.Name()) {
			return fs.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if strings.HasPrefix(imp.Path.Value, `"github.com/wailsapp/wails/v3/pkg/application`) {
				checked++
				if f.Name.Name != "main" {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s (package %s) imports Wails application", rel, f.Name.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("found no Wails import; the scan is looking in the wrong place")
	}
}
