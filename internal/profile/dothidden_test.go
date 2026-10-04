package profile

import (
	"testing"

	"github.com/Rethunk-AI/mortar/internal/settings"
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
	if _, err := e.SetModEnabled("stardew", p.ID, "pack", "me.off", false); err != nil {
		t.Fatal(err)
	}
	if got, err := svc.DotHiddenMods("stardew", p.ID); err != nil || len(got) != 0 {
		t.Fatalf("setting off = %+v, %v", got, err)
	}
	setGamePref(t, st, "showDotHiddenMods", "true")
	got, err := svc.DotHiddenMods("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"me.hidden": ".Hidden", "me.deep": ".Group/Deep"}
	if len(got) != len(want) {
		t.Fatalf("hidden = %+v", got)
	}
	for _, h := range got {
		if h.Key != "pack" || want[h.UniqueID] != h.Folder {
			t.Errorf("unexpected %+v", h)
		}
	}
}
