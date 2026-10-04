package profile

import (
	"path/filepath"
	"testing"
)

func TestMoveGameModsMovesNewFoldersAndSkipsHeldOnes(t *testing.T) {
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
	if _, err := e.MoveGameMods("stardew", p.ID, game, []string{"../escape"}); err == nil {
		t.Fatal("moved a folder outside the Mods folder")
	}
	res, err := e.MoveGameMods("stardew", p.ID, game, []string{filepath.Join(game, "New"), ".Off", "Held"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 2 || res.Skipped != 1 || len(res.Profile.Entries) != 3 {
		t.Fatalf("result = %+v", res)
	}
	if got := names(t, game); len(got) != 1 || got[0] != "Held" {
		t.Fatalf("game Mods left = %v", got)
	}
	cfg, err := e.ModFolder("stardew", p.ID, "", "me.new")
	if err != nil || read(t, filepath.Join(cfg, "config.json")) != `{"x":1}` {
		t.Fatalf("config not carried: %v", err)
	}
	for _, en := range res.Profile.Entries {
		if en.Mods[0].UniqueID == "me.off" && len(en.Disabled) != 1 {
			t.Errorf("dotted folder arrived switched on: %+v", en)
		}
	}
	hist, err := e.History("stardew", p.ID)
	if err != nil || len(hist) == 0 || hist[0].Kind != historyImported {
		t.Fatalf("history = %+v, %v", hist, err)
	}
}
