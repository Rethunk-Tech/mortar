package stardew

import (
	"path/filepath"
	"testing"
)

func TestSteamLaunchWithLoaderKeepsTheUsersOptions(t *testing.T) {
	dir := filepath.Join("games", "Stardew Valley")
	exe := `"` + filepath.Join(dir, "StardewModdingAPI.exe") + `"`
	for current, want := range map[string]string{
		"":                        exe + " %command%",
		"-windowed":               exe + " %command% -windowed",
		"DXVK_HUD=1 %command% -x": "DXVK_HUD=1 " + exe + " %command% -x",
		exe + " %command%":        exe + " %command%",
	} {
		if got := (Game{}).SteamLaunchWithLoader(dir, current); got != want {
			t.Errorf("%q -> %q, want %q", current, got, want)
		}
	}
}
