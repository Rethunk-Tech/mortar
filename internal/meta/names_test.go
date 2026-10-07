package meta_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// keptCacheFiles are cached names Clean up deliberately leaves alone: the compatibility list serves offline and the
// components manifest is replaced in place on each build.
var keptCacheFiles = map[string]bool{"CompatCacheFile": true, "cacheName": true}

// nameBuilders build a cache name from a names.go prefix inside their own package.
var nameBuilders = map[string]bool{"DetailsName": true, "PageName": true}

func isCacheCall(call *ast.CallExpr) bool {
	var name string
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = fn.Sel.Name
	case *ast.Ident:
		name = fn.Name
	case *ast.IndexExpr:
		return isCacheCall(&ast.CallExpr{Fun: fn.X})
	}
	return name == "Cached" || name == "CachedForBuild"
}

// registered reports whether the cache name expression starts from a names.go prefix, a builder of one, or a kept file.
func registered(arg ast.Expr, prefixes map[string]bool) bool {
	switch e := arg.(type) {
	case *ast.BinaryExpr:
		return registered(e.X, prefixes)
	case *ast.CallExpr:
		switch fn := e.Fun.(type) {
		case *ast.Ident:
			return nameBuilders[fn.Name]
		case *ast.SelectorExpr:
			if x, ok := fn.X.(*ast.Ident); ok && x.Name == "fmt" && fn.Sel.Name == "Sprintf" && len(e.Args) > 1 {
				if lit, ok := e.Args[0].(*ast.BasicLit); ok && lit.Value == `"%s%s-%d.json"` {
					return registered(e.Args[1], prefixes)
				}
				return false
			}
			return nameBuilders[fn.Sel.Name]
		}
	case *ast.Ident:
		return prefixes[e.Name] || keptCacheFiles[e.Name]
	case *ast.SelectorExpr:
		return prefixes[e.Sel.Name] || keptCacheFiles[e.Sel.Name]
	}
	return false
}

func namesPrefixes(t *testing.T) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "names.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, d := range file.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok {
					for _, n := range vs.Names {
						out[n.Name] = true
					}
				}
			}
		}
	}
	return out
}

// TestEveryCacheWriteIsRegistered fails for a Cached call whose file name does not come from a names.go prefix, so a
// new cache cannot ship without an expiry rule in Clean up (internal/datasvc/cleanup.go).
func TestEveryCacheWriteIsRegistered(t *testing.T) {
	prefixes := namesPrefixes(t)
	var bad []string
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, readErr := fsx.ReadFile(path)
		if readErr != nil || !strings.Contains(string(src), "Cached") {
			return readErr
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if ok && isCacheCall(call) && len(call.Args) >= 4 && !registered(call.Args[1], prefixes) {
				bad = append(bad, fset.Position(call.Pos()).String())
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) > 0 {
		t.Fatalf("cache names not built from a meta/names.go prefix (add one and a rule in datasvc.cacheTTL): %v", bad)
	}
}

func TestChangelogsHaveAnExpiryRule(t *testing.T) {
	if !namesPrefixes(t)["NexusChangelogsPrefix"] {
		t.Fatal("NexusChangelogsPrefix is not in names.go")
	}
}
