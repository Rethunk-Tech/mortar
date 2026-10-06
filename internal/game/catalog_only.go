package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// catalogOnly is a game its catalog entry describes in full: names, store ids, sources and the marker file.
type catalogOnly string

func (g catalogOnly) ID() string { return string(g) }

func (g catalogOnly) Name() string {
	info, _ := catalogGame(string(g))
	return info.Name
}

func (g catalogOnly) SteamAppID() string {
	info, _ := catalogGame(string(g))
	return info.SteamAppID()
}

func (g catalogOnly) ModSources() []string {
	info, _ := catalogGame(string(g))
	ids := make([]string, len(info.Sources))
	for i, s := range info.Sources {
		ids[i] = s.ID
	}
	return ids
}

// GameProcesses are the marker, when it is an executable, and the marker without its extension: a Windows build's
// process under Proton, and a native build's.
func (g catalogOnly) GameProcesses() []string {
	info, _ := catalogGame(string(g))
	stem := strings.TrimSuffix(info.Marker, filepath.Ext(info.Marker))
	if strings.EqualFold(filepath.Ext(info.Marker), ".exe") {
		return []string{info.Marker, stem}
	}
	return []string{stem}
}

// ValidInstall reports why dir is not the game's install folder: it lacks the catalog's marker file.
func (g catalogOnly) ValidInstall(dir string) error {
	info, _ := catalogGame(string(g))
	st, err := os.Stat(filepath.Join(dir, info.Marker))
	if err != nil || !st.Mode().IsRegular() {
		return fmt.Errorf("%q is not a %s folder: it has no %s", dir, info.Name, info.Marker)
	}
	return nil
}

// stardewValley differs from its catalog entry only in its processes: SMAPI's install renames and wraps the game's
// launcher, and those names must block a loader install too.
type stardewValley struct{ catalogOnly }

func (stardewValley) GameProcesses() []string {
	return []string{"Stardew Valley", "StardewValley", "StardewValley-original", "StardewValley.bin.x86_64"}
}
