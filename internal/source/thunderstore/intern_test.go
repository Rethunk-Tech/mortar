package thunderstore

import (
	"slices"
	"testing"
)

func TestDepCompactorSharesIdenticalListsAndKeepsTheDependencies(t *testing.T) {
	t.Parallel()
	deps := []string{"BepInEx-BepInExPack-5.4.2100", "Owner-Lib-1.0.0"}
	mk := func(owner string) pkg {
		return pkg{Owner: owner, Versions: []version{{Number: "1.0.0", Deps: slices.Clone(deps)}, {Number: "1.0.1", Deps: slices.Clone(deps)}}}
	}
	c := newDepCompactor()
	a, b := mk("A"), mk("B")
	c.add(&a)
	c.add(&b)
	if &a.Versions[0].ids[0] != &b.Versions[1].ids[0] {
		t.Error("identical dependency lists must share one id slice")
	}
	if got := a.depsOf(a.Versions[1]); !slices.Equal(got, deps) {
		t.Errorf("deps = %v, want %v", got, deps)
	}
}
