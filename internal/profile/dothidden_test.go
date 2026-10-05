package profile

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestDotHiddenModsListsOnlyDotsMortarDidNotAdd(t *testing.T) {
	e := newEnv(t)
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	e.item(t, "pack", map[string]string{
		"On/manifest.json":          manifestJSON("me.on"),
		"Off/manifest.json":         manifestJSON("me.off"),
		".Hidden/manifest.json":     manifestJSON("me.hidden"),
		".Group/Deep/manifest.json": manifestJSON("me.deep"),
	})
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "pack", Source{Kind: KindLocal, Name: "pack.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, "pack", "smapi:me.off", false); err != nil {
		t.Fatal(err)
	}
	if got, err := svc.DotHiddenMods("stardew", p.ID, ""); err != nil || len(got) != 0 {
		t.Fatalf("setting off = %+v, %v", got, err)
	}
	setGamePref(t, st, "showDotHiddenMods", "true")
	got, err := svc.DotHiddenMods("stardew", p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"me.hidden": ".Hidden", "me.deep": ".Group/Deep"}
	if len(got) != len(want) {
		t.Fatalf("hidden = %+v", got)
	}
	for _, h := range got {
		if h.Key != "pack" || want[h.ID.Local()] != h.Folder {
			t.Errorf("unexpected %+v", h)
		}
	}
	if one, err := svc.DotHiddenMods("stardew", p.ID, "pack"); err != nil || len(one) != len(want) {
		t.Fatalf("keyed = %+v, %v", one, err)
	}
	if none, err := svc.DotHiddenMods("stardew", p.ID, "other"); err != nil || len(none) != 0 {
		t.Fatalf("other key = %+v, %v", none, err)
	}
}

func TestUnhideModRenamesDottedFoldersAndRegistersTheMod(t *testing.T) {
	e := newEnv(t)
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(e.Store, t.TempDir(), st)
	e.item(t, "pack", map[string]string{
		"On/manifest.json":          manifestJSON("me.on"),
		".Group/Deep/manifest.json": manifestJSON("me.deep"),
	})
	p, err := e.Create("stardew", "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.AddEntry("stardew", p.ID, "pack", Source{Kind: KindLocal, Name: "pack.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UnhideMod("stardew", p.ID, "pack", "Nope"); err == nil {
		t.Fatal("unknown folder accepted")
	}
	shown, err := svc.store.hiddenModPath("stardew", p.ID, "pack", ".Group/Deep")
	if err != nil || !exists(shown) {
		t.Fatalf("hidden path = %q, %v", shown, err)
	}
	got, err := svc.UnhideMod("stardew", p.ID, "pack", ".Group/Deep")
	if err != nil {
		t.Fatal(err)
	}
	var folders []string
	for _, m := range got.Entries[0].Mods {
		folders = append(folders, m.Folder)
	}
	if !slices.Contains(folders, "Group/Deep") || exists(shown) {
		t.Fatalf("mods = %v, old path exists = %v", folders, exists(shown))
	}
	if left, err := e.DotHiddenMods("stardew", p.ID, ""); err != nil || len(left) != 0 {
		t.Fatalf("still hidden: %+v, %v", left, err)
	}
}
