package game

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/customgame"
	"github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
)

func TestACustomGameJoinsTheCatalogForEveryLoaderAndStaysLocal(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { components.SetUserGames(nil) })
	install := t.TempDir()
	exe := filepath.Join(install, "My Game.exe")
	if err := os.WriteFile(exe, []byte("MZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, loader := range []string{customgame.BepInEx5, customgame.BepInEx6, customgame.MelonLoader, customgame.Shimloader, customgame.Folder} {
		e, dir, err := customgame.Add("My Game", exe, loader, "my-game")
		if err != nil || dir != install || e.Exe != "My Game.exe" {
			t.Fatalf("%s: %+v, %q, %v", loader, e, dir, err)
		}
		g := Find(e.ID)
		if g == nil || g.Name() != "My Game" || g.ValidInstall(install) != nil || !slices.Contains(Implemented(), e.ID) {
			t.Fatalf("%s: the game %q is not set up like a catalog game", loader, e.ID)
		}
		if len(Loaders(e.ID)) == 0 {
			t.Errorf("%s: no loader driver answers for %q", loader, e.ID)
		}
	}
	games, err := customgame.List()
	if err != nil || len(games) != 5 || games[1].ID != "custom-my-game-2" {
		t.Fatalf("saved %+v, %v", games, err)
	}
	// A later start reads them back from the file.
	components.SetUserGames(nil)
	if err := customgame.Load(); err != nil || Find("custom-my-game") == nil {
		t.Fatalf("load: %v", err)
	}
	if _, _, err := customgame.Add("Stardew Valley", exe, "nonsense", ""); err == nil {
		t.Error("an unknown loader was accepted")
	}
	if err := customgame.Remove("custom-my-game"); err != nil || Find("custom-my-game") != nil {
		t.Errorf("remove: %v", err)
	}
	if _, _, err := customgame.Add("No Community", exe, customgame.Shimloader, ""); err == nil {
		t.Error("a Shimloader game without a Thunderstore community was accepted")
	}
	// A custom Shimloader game lays a mod out as a catalog one does and places its files beside the chosen executable.
	if err := customgame.Load(); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"custom-my-game-4", "palworld"} {
		info, _ := components.Game(id)
		got := thunderpack.Layout("shimloader", []string{"mod/a.lua", "pak/b.pak", "cfg/c.cfg"}, "Ns-Mod", info.Loaders[0].Routes)
		if got["mod/a.lua"] != "shimloader/mod/Ns-Mod/a.lua" || got["pak/b.pak"] != "shimloader/pak/Ns-Mod/b.pak" || got["cfg/c.cfg"] != "shimloader/cfg/c.cfg" {
			t.Errorf("%s: layout %v", id, got)
		}
	}
	if info, _ := components.Game("custom-my-game-4"); info.LoaderPlaceDir("shimloader") != "" {
		t.Error("a custom game's files must sit beside the executable it chose")
	}
	// The catalog's own games are untouched by all of it.
	if g := Find("lethal-company"); g == nil {
		t.Error("a catalog game vanished")
	}
}
