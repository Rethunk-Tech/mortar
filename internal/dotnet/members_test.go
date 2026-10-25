package dotnet

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// testdata/mod.dll: an instance method with two parameters, a static one, and no compiler-generated members (its
// lambdas and the <Module> type).
func TestMembersListsMethodsWithParameters(t *testing.T) {
	got, err := Members(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	find := func(typ, method string) Member {
		i := slices.IndexFunc(got, func(m Member) bool { return m.Type == typ && m.Method == method })
		if i < 0 {
			t.Fatalf("no %s::%s in %v", typ, method, got)
		}
		return got[i]
	}
	if m := find("Fixture.Mod", "own"); m.Kind != "field" || m.Static {
		t.Errorf("own = %+v", m)
	}
	if m := find("Fixture.Mod", "get_Own"); m.Void || m.Kind != "method" {
		t.Errorf("a getter returns its value: %+v", m)
	}
	if m := find("Fixture.Mod", "set_Own"); !m.Void {
		t.Errorf("a setter returns nothing: %+v", m)
	}
	if m := find("Fixture.Mod", "SwitchTool"); m.Static || !slices.Equal(m.Params, []string{"farmer", "kind"}) {
		t.Errorf("SwitchTool = %+v", m)
	}
	if m := find("Fixture.Patches.DrawPatch", "Transpiler"); !m.Static || !slices.Equal(m.Params, []string{"code"}) {
		t.Errorf("Transpiler = %+v", m)
	}
	for _, m := range got {
		if strings.Contains(m.Type+m.Method, "<") {
			t.Errorf("compiler-generated %+v", m)
		}
	}
}
