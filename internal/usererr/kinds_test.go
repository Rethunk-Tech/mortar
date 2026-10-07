package usererr

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"testing"
)

func TestKindsListsEveryKindConstant(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "usererr.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var declared []Kind
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, s := range gd.Specs {
			vs, ok := s.(*ast.ValueSpec)
			if !ok {
				continue
			}
			if id, ok := vs.Type.(*ast.Ident); ok && id.Name == "Kind" {
				for _, v := range vs.Values {
					if lit, ok := v.(*ast.BasicLit); ok {
						declared = append(declared, Kind(lit.Value[1:len(lit.Value)-1]))
					}
				}
			}
		}
	}
	if len(declared) == 0 {
		t.Fatal("found no Kind constants")
	}
	for _, k := range declared {
		if !slices.Contains(Kinds, k) {
			t.Errorf("Kinds is missing %q", k)
		}
	}
}
