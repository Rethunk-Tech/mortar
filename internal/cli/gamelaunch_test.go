package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/control"
)

func TestGameSteamLaunchOption(t *testing.T) {
	r := invoke(t, map[string]any{
		"game.steamLaunchOption": control.SteamLaunchOptionResult{Options: "old"},
	}, "game", "steam-launch-option", "stardew")
	if r.code != 0 || r.calls[0].method != "game.steamLaunchOption" || !strings.Contains(r.out, "old") {
		t.Fatalf("read: %+v %q", r, r.out)
	}
	r = invoke(t, map[string]any{
		"game.steamLaunchOption": control.SteamLaunchOptionResult{Options: "new", Set: true},
	}, "game", "steam-launch-option", "stardew", "--set")
	if r.code != 0 || !r.calls[0].params.Set || !strings.Contains(r.out, "new") {
		t.Fatalf("set: %+v %q", r, r.out)
	}
	r = invoke(t, nil, "game", "steam-launch-option", "stardew", "--set=1")
	if r.code != 2 || len(r.calls) != 0 || !strings.Contains(r.errOut, "takes no value") {
		t.Fatalf("--set=: %+v %q", r, r.errOut)
	}
}

func TestGameFolderSendsAnAbsolutePathAndPrintsTheStoredOne(t *testing.T) {
	r := invoke(t, map[string]any{
		"game.folder": control.GameFolderResult{Game: "stardew", Folder: "/games/Stardew"},
	}, "game", "folder", "stardew", "rel/dir")
	if r.code != 0 || r.calls[0].method != "game.folder" || r.calls[0].params.Game != "stardew" ||
		!filepath.IsAbs(r.calls[0].params.Path) || !strings.HasSuffix(r.calls[0].params.Path, filepath.FromSlash("rel/dir")) ||
		!strings.Contains(r.out, "/games/Stardew") {
		t.Fatalf("folder: %+v %q", r, r.out)
	}
	if r = invoke(t, nil, "game", "folder", "stardew"); r.code != 2 || len(r.calls) != 0 {
		t.Fatalf("missing folder: %+v", r)
	}
}
