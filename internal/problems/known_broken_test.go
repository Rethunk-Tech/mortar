package problems

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestCuratedBrokenNamesPackagesAndModsInTheVersionRange(t *testing.T) {
	t.Parallel()
	list := []components.KnownBroken{
		{ID: "Ns-OldLib", Versions: "<2.1.0", Reason: "Crashes on load with the current game.", Replacement: "Ns-NewLib"},
		{ID: "Ns-Anything", Reason: "Never worked."},
		{ID: "Ns-Ranged", Versions: ">=1.0.0 <1.5.0", Reason: "Breaks saves."},
	}
	pkgs := []profile.PackageRef{
		{Key: "a", ID: mod.ID("thunderstore:Ns-OldLib"), Name: "Ns-OldLib", Version: "2.0.0"},
		{Key: "b", ID: mod.ID("thunderstore:Ns-Anything"), Name: "Ns-Anything", Version: "9.9.9"},
		{Key: "c", ID: mod.ID("thunderstore:Ns-Ranged"), Name: "Ns-Ranged", Version: "1.5.0"},
	}
	mods := []framework.Mod{{Key: "d", Name: "Ns-Ranged", UniqueID: "Ns-Ranged", Version: "1.2.0"}}
	got := curatedBroken(list, "", pkgs, mods)
	keys := map[string]Broken{}
	for _, b := range got {
		keys[b.Key] = b
	}
	if len(got) != 3 || keys["a"].Replacement == nil || keys["a"].Replacement.PageName != "Ns-NewLib" || keys["c"].Key != "" {
		t.Fatalf("rows = %+v", got)
	}
	for _, b := range got {
		if b.Status != "broken" || b.Source != "Mortar" || b.Summary == "" {
			t.Errorf("row %+v", b)
		}
	}
	if _, ok := keys["d"]; !ok {
		t.Errorf("the mod in range was not flagged: %+v", got)
	}
}

func TestInVersionRange(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		version, versions string
		want              bool
	}{
		{"1.0.0", "", true},
		{"1.0.0", "<2.0.0", true},
		{"2.0.0", "<2.0.0", false},
		{"2.0.0", "<=2.0.0", true},
		{"1.2.0", ">=1.0.0 <1.5.0", true},
		{"1.5.0", ">=1.0.0 <1.5.0", false},
		{"3.0.0", "=3.0.0", true},
		{"not-a-version", "<2.0.0", false},
	} {
		if got := inVersionRange(c.version, c.versions); got != c.want {
			t.Errorf("inVersionRange(%q, %q) = %v, want %v", c.version, c.versions, got, c.want)
		}
	}
}
