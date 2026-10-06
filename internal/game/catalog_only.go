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
// process under Proton, and a native build's; and the native Linux build's executable when the catalog names one.
func (g catalogOnly) GameProcesses() []string {
	info, _ := catalogGame(string(g))
	stem := strings.TrimSuffix(info.Marker, filepath.Ext(info.Marker))
	out := []string{stem}
	if strings.EqualFold(filepath.Ext(info.Marker), ".exe") {
		out = []string{info.Marker, stem}
	}
	if info.LinuxMarker != "" {
		out = append(out, info.LinuxMarker)
	}
	return out
}

// ValidInstall reports why dir is not the game's install folder: it lacks the catalog's marker file and its native
// Linux build's executable.
func (g catalogOnly) ValidInstall(dir string) error {
	info, _ := catalogGame(string(g))
	for _, m := range []string{info.Marker, info.LinuxMarker} {
		if st, err := os.Stat(filepath.Join(dir, m)); m != "" && err == nil && st.Mode().IsRegular() {
			return nil
		}
	}
	return fmt.Errorf("%q is not a %s folder: it has no %s", dir, info.Name, info.Marker)
}

// stardewValley differs from its catalog entry only in its processes: SMAPI's install renames and wraps the game's
// launcher, and those names must block a loader install too.
type stardewValley struct{ catalogOnly }

func (stardewValley) GameProcesses() []string {
	return []string{"Stardew Valley", "StardewValley", "StardewValley-original", "StardewValley.bin.x86_64"}
}
