package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

func TestPackageEntriesTakeTheirIconAndCategoryFromTheCachedIndex(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create("lethal-company", "LC")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.InstallSource(t.Context(), "lethal-company", p.ID, tsZip(t, "1.0.0"), Source{Kind: KindThunderstore, Name: "Ns-Mod", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	icon := "https://ccdn.thunderstore.io/live/repository/icons/Ns-Mod-1.0.0.png"
	asked := 0
	e.PackageLooks = func(game string, names []string) map[string]thunderstore.Look {
		asked++
		if game != "lethal-company" || len(names) != 1 || names[0] != "Ns-Mod" {
			t.Errorf("asked %s %v", game, names)
		}
		return map[string]thunderstore.Look{"ns-mod": {Icon: icon, Category: "Tools"}}
	}
	if err := e.FillPackageLooks("lethal-company"); err != nil {
		t.Fatal(err)
	}
	mods, err := e.Mods("lethal-company", p.ID)
	if err != nil || len(mods) != 1 || mods[0].Picture != icon {
		t.Fatalf("mods = %+v, %v", mods, err)
	}
	got, _ := e.read("lethal-company", p.ID)
	if got.Entries[0].Source.Category != "Tools" {
		t.Fatalf("source = %+v", got.Entries[0].Source)
	}
	if err := e.FillPackageLooks("lethal-company"); err != nil || asked != 1 {
		t.Fatalf("a filled profile asked the index again: %d, %v", asked, err)
	}
}
