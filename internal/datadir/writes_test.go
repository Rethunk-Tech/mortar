package datadir

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// writersOutsideDatadir are the non-test files that create files directly. State under the data folder goes through
// WriteFileAtomic, WriteJSON or fsx so a crash never leaves it truncated; these write something else.
var writersOutsideDatadir = map[string]string{
	"components/components.go": "temp file beside the catalog cache, renamed over it",
	"launch/output.go":         "a throwaway game log in the OS temp folder",
	"sampler/ipc.go":           "a throwaway trace in the OS temp folder",
	"cli/console_windows.go":   "opens the console device, creates nothing",
}

// TestFilesAreCreatedOnlyThroughDatadirOrFsx fails when non-test code under internal/ calls os.WriteFile, os.Create,
// os.CreateTemp or os.OpenFile outside datadir, fsx and the files above.
func TestFilesAreCreatedOnlyThroughDatadirOrFsx(t *testing.T) {
	t.Parallel()
	creators := map[string]bool{"WriteFile": true, "Create": true, "CreateTemp": true, "OpenFile": true}
	seen := map[string]bool{}
	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel := filepath.ToSlash(strings.TrimPrefix(path, ".."+string(filepath.Separator)))
		if strings.HasPrefix(rel, "datadir/") || strings.HasPrefix(rel, "fsx/") || strings.Contains(rel, "/testdata/") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !creators[sel.Sel.Name] {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "os" {
				return true
			}
			if _, ok := writersOutsideDatadir[rel]; ok {
				seen[rel] = true
				return true
			}
			t.Errorf("%s: os.%s creates a file; write state through datadir.WriteFileAtomic or fsx, or list the file in writersOutsideDatadir with why", fset.Position(call.Pos()), sel.Sel.Name)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for rel := range writersOutsideDatadir {
		if !seen[rel] {
			t.Errorf("%s no longer creates files directly; drop it from writersOutsideDatadir", rel)
		}
	}
}
