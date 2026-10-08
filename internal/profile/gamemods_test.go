package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestMoveGameModsMovesNewFoldersAndSkipsHeldOnes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	game := t.TempDir()
	writeFile(t, game, "New/manifest.json", manifestJSON("me.new"))
	writeFile(t, game, "New/config.json", `{"x":1}`)
	writeFile(t, game, ".Off/manifest.json", manifestJSON("me.off"))
	writeFile(t, game, "Held/manifest.json", manifestJSON("me.held"))
	e.item(t, "held", map[string]string{"manifest.json": manifestJSON("me.held")})
	p := mustCreate(t, e, "P")
	if _, err := e.AddEntry("stardew", p.ID, "held", Source{Kind: KindLocal, Name: "held.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.MoveGameMods(t.Context(), "stardew", p.ID, game, []string{"../escape"}); err == nil {
		t.Fatal("moved a folder outside the Mods folder")
	}
	res, err := e.MoveGameMods(t.Context(), "stardew", p.ID, game, []string{filepath.Join(game, "New"), ".Off", "Held"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 2 || res.Skipped != 1 || len(res.Profile.Entries) != 3 {
		t.Fatalf("result = %+v", res)
	}
	if got := names(t, game); len(got) != 1 || got[0] != "Held" {
		t.Fatalf("game Mods left = %v", got)
	}
	cfg, err := e.ModFolder("stardew", p.ID, "", "smapi:me.new")
	if err != nil || read(t, filepath.Join(cfg, "config.json")) != `{"x":1}` {
		t.Fatalf("config not carried: %v", err)
	}
	for _, en := range res.Profile.Entries {
		if en.Mods[0].ID == "smapi:me.off" && len(en.Disabled) != 1 {
			t.Errorf("dotted folder arrived switched on: %+v", en)
		}
	}
	hist, err := e.History("stardew", p.ID)
	if err != nil || len(hist) == 0 || hist[0].Kind != historyImported {
		t.Fatalf("history = %+v, %v", hist, err)
	}
}

func modAt(id, version string) string {
	return `{"Name":"` + id + `","Author":"me","Version":"` + version + `","UniqueID":"` + id + `"}`
}

func TestGameModsDiffTracksWhatTheProfileLacks(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	game := t.TempDir()
	writeFile(t, game, "A/manifest.json", modAt("me.a", "1.0.0"))
	writeFile(t, game, "B/manifest.json", modAt("me.b", "1.0.0"))
	writeFile(t, game, "Bad/manifest.json", "{not json")
	writeFile(t, game, "Quiet/manifest.json", modAt("me.quiet", "1.0.0"))
	p := mustCreate(t, e, "P")
	folders := []string{"A", "B", "Quiet"}
	if _, err := e.SyncGameMods(t.Context(), "stardew", p.ID, game, folders, false, false); err != nil {
		t.Fatal(err)
	}
	diff := func() GameModsDiff {
		t.Helper()
		d, err := e.GameModsDiff("stardew", p.ID, game, []string{"Quiet"})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	d := diff()
	if len(d.Missing) != 0 || len(d.Different) != 0 || d.Same != 2 {
		t.Fatalf("fully imported folder differs: %+v", d)
	}
	if len(d.Unreadable) != 1 || d.Unreadable[0].Reason == "" {
		t.Fatalf("unreadable = %+v", d.Unreadable)
	}

	if _, err := e.RemoveEntry("stardew", p.ID, mustKey(t, e, p.ID, "smapi:me.b")); err != nil {
		t.Fatal(err)
	}
	if d = diff(); len(d.Missing) != 1 || d.Missing[0].Name != "me.b" || filepath.Base(d.Missing[0].Folder) != "B" {
		t.Fatalf("removed mod not missing: %+v", d)
	}

	writeFile(t, game, "A/manifest.json", modAt("me.a", "1.2.0"))
	d = diff()
	if len(d.Different) != 1 || d.Different[0].Newer != "folder" || d.Different[0].ProfileVersion != "1.0.0" {
		t.Fatalf("newer folder copy: %+v", d)
	}
	res, err := e.SyncGameMods(t.Context(), "stardew", p.ID, game, []string{"A", "B"}, false, true)
	if err != nil || res.Imported != 2 {
		t.Fatalf("sync = %+v, %v", res, err)
	}
	if d = diff(); len(d.Missing) != 0 || len(d.Different) != 0 {
		t.Fatalf("after sync: %+v", d)
	}
}

func mustKey(t *testing.T, e env, id string, modID mod.ID) string {
	t.Helper()
	p, err := e.load("stardew", id)
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range p.Entries {
		if en.Mods[0].ID == modID {
			return en.Key
		}
	}
	t.Fatalf("no entry holds %s", modID)
	return ""
}
