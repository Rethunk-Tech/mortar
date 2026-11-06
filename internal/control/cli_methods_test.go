package control

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// computedCLIMethods are the names the CLI builds as prefix+subcommand, so no literal holds them. Each list is the
// subcommand set of the switch that builds the name in internal/cli.
var computedCLIMethods = map[string][]string{
	"mods.": {
		"enable", "disable", "remove", "pin", "unpin", "tag", "untag", "category", "note", "skip-version", "split", "combine",
		"favourite", "unfavourite", "favourites",
	},
	"profile.": {"freeze", "unfreeze", "favourite", "unfavourite"},
	"queue.":   {"retry", "skip", "pause-item", "resume-item", "top", "up", "down", "pause", "resume", "clear"},
	"backups.": {"keep", "unkeep"},
}

// cliLiteralMethods returns the method names internal/cli passes as string literals: the method argument of
// c.call, show and installMethod, and the method fields of the table of verbs that map straight onto a method.
func cliLiteralMethods(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	files, err := filepath.Glob("../cli/*.go")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	lit := func(e ast.Expr) {
		if b, ok := e.(*ast.BasicLit); ok && b.Kind == token.STRING {
			if s, err := strconv.Unquote(b.Value); err == nil {
				found[s] = true
			}
		}
	}
	for _, fname := range files {
		if strings.HasSuffix(fname, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, fname, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		{
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CallExpr:
					fun := n.Fun
					if ix, ok := fun.(*ast.IndexExpr); ok {
						fun = ix.X
					}
					switch fn := fun.(type) {
					case *ast.SelectorExpr:
						if (fn.Sel.Name == "call" || fn.Sel.Name == "installMethod") && len(n.Args) > 0 {
							lit(n.Args[0])
						}
					case *ast.Ident:
						if fn.Name == "show" && len(n.Args) > 1 {
							lit(n.Args[1])
						}
					}
				case *ast.KeyValueExpr:
					if k, ok := n.Key.(*ast.Ident); ok && (k.Name == "method" || k.Name == "method2") {
						lit(n.Value)
					}
				}
				return true
			})
		}
	}
	out := make([]string, 0, len(found))
	for m := range found {
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

func TestEveryMethodTheCLISendsIsDispatched(t *testing.T) {
	methods := cliLiteralMethods(t)
	if len(methods) < 70 {
		t.Fatalf("found only %d literal CLI methods; the scan no longer matches how the CLI calls", len(methods))
	}
	for prefix, subs := range computedCLIMethods {
		for _, sub := range subs {
			methods = append(methods, prefix+sub)
		}
	}
	s := services(t)
	for _, m := range methods {
		if unrouted(s, m) {
			t.Errorf("the CLI sends %q and no handler answers it", m)
		}
	}
}

// unrouted reports whether Handle answers method as unknown. A handler that runs far enough to reach a service this
// bare fixture leaves nil panics, which also proves it was dispatched.
func unrouted(s *Services, method string) (unknown bool) {
	defer func() { _ = recover() }()
	_, err := s.Handle(context.Background(), method, Params{})
	return err != nil && strings.HasPrefix(err.Error(), "unknown method")
}
