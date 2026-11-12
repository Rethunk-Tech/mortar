package game

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
)

// catalogOnly is a game its catalog entry describes in full: names, store ids, sources and the marker file.
type catalogOnly string

func (g catalogOnly) ID() string { return string(g) }

func (g catalogOnly) Name() string {
	info, _ := components.Game(string(g))
	return info.Name
}

func (g catalogOnly) SteamAppID() string {
	info, _ := components.Game(string(g))
	return info.SteamAppID()
}

func (g catalogOnly) ModSources() []string {
	info, _ := components.Game(string(g))
	ids := make([]string, len(info.Sources))
	for i, s := range info.Sources {
		ids[i] = s.ID
	}
	return ids
}

// GameProcesses are the marker, when it is an executable, and the marker without its extension: a Windows build's
// process under Proton, and a native build's; the native Linux build's executable when the catalog names one; and the
// other executables the catalog lists.
func (g catalogOnly) GameProcesses() []string {
	info, _ := components.Game(string(g))
	stem := strings.TrimSuffix(info.Marker, filepath.Ext(info.Marker))
	out := []string{stem}
	if strings.EqualFold(filepath.Ext(info.Marker), ".exe") {
		out = []string{info.Marker, stem}
	}
	if info.LinuxMarker != "" {
		out = append(out, path.Base(info.LinuxMarker))
	}
	if info.DarwinMarker != "" {
		out = append(out, path.Base(info.DarwinMarker))
	}
	for _, p := range info.Processes {
		out = append(out, p, strings.TrimSuffix(p, filepath.Ext(p)))
	}
	return out
}

// ValidInstall reports why dir is not the game's install folder: it lacks the catalog's marker file and its native
// Linux build's executable.
func (g catalogOnly) ValidInstall(dir string) error {
	info, _ := components.Game(string(g))
	for _, m := range []string{info.MarkerPath(), filepath.FromSlash(info.LinuxMarker), filepath.FromSlash(info.DarwinMarker), info.Stores.StoreMarker("gog"), info.Stores.StoreMarker("epic"), info.Stores.StoreMarker("xbox")} {
		if m != "" && safefs.IsFile(filepath.Join(dir, m)) {
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
