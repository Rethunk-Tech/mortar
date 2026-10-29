package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/metadata"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/shots"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Catalog path roles.
const (
	PathSaves              = "saves"
	PathStartupPreferences = "startupPreferences"
	// PathLog is the game's own log file, which the Console follows when the loader keeps none (Vintage Story's
	// client-main.log), and PathCrashLog the file it writes a crash report to.
	PathLog      = "log"
	PathCrashLog = "crashLog"
	// PathScreenshots is the folder the game saves its own screenshots to.
	PathScreenshots = "screenshots"
)

// HasScreenshots reports whether game id has a screenshots folder of its own or is a Steam game, whose screenshots
// Steam keeps.
func HasScreenshots(id string) bool {
	info, ok := catalogGame(id)
	return ok && (HasPath(id, PathScreenshots) || info.SteamAppID() != "")
}

// ScreenshotDirs is game id's screenshot folders by source: its own folder, and Steam's for a Steam game when Steam
// is installed.
func ScreenshotDirs(home string, s settings.Settings, id string) map[string]string {
	out := map[string]string{}
	if HasPath(id, PathScreenshots) {
		if dir, err := PathFor(home, s, id, "", PathScreenshots); err == nil {
			out[shots.SourceGame] = dir
		}
	}
	info, _ := catalogGame(id)
	if app := info.SteamAppID(); app != "" {
		if st, status := LocateSteam(home, s); status == steam.Found {
			if dir, err := st.ScreenshotsDir(app); err == nil {
				out[shots.SourceSteam] = dir
			}
		}
	}
	return out
}

// HasPath reports whether the catalog gives game id a path for role: a game has a save folder or startup
// preferences when its entry says where they live.
func HasPath(id, role string) bool {
	g, ok := catalogGame(id)
	_, has := g.Paths[role]
	return ok && has
}

// HasSaves reports whether game id has a save folder.
func HasSaves(id string) bool { return HasPath(id, PathSaves) }

// HasStartupSettings reports whether game id reads startup preferences from a file outside the install.
func HasStartupSettings(id string) bool { return HasPath(id, PathStartupPreferences) }

// PathFor resolves game id's path for role in the install pin names (the selected install when empty).
func PathFor(home string, s settings.Settings, id, pin, role string) (string, error) {
	g, ok := catalogGame(id)
	t, has := g.Paths[role]
	if !ok || !has {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no %s path", id, role))
	}
	return PathIn(home, s, id, pin, t)
}

// PathIn resolves the path template t for game id's install pin names (the selected install when empty), inside its
// Wine or Proton prefix when it runs in one.
func PathIn(home string, s settings.Settings, id, pin string, t components.PathTemplate) (string, error) {
	g, _ := catalogGame(id)
	in, err := ResolveInstall(home, s, id, pin)
	if err != nil {
		return "", err
	}
	p, err := runtime.Resolve(in.runtime(g, home), t)
	if err != nil {
		return "", err
	}
	return localizedFolder(p, g.FolderNames), nil
}

// localizedFolder swaps the first folder of p that names lists other names for to whichever of them exists and changed
// last, so a game that names its folder after its language is found in the language in use.
// ponytail: "changed last" is the folder's own time, which moves when the game adds or removes a file in it.
func localizedFolder(p string, names map[string][]string) string {
	sep := string(filepath.Separator)
	parts := strings.Split(p, sep)
	for i, part := range parts {
		alts, ok := names[part]
		if !ok || i == 0 {
			continue
		}
		parent := strings.Join(parts[:i], sep)
		var newest time.Time
		for _, name := range append([]string{part}, alts...) {
			if fi, err := os.Stat(parent + sep + name); err == nil && fi.IsDir() && fi.ModTime().After(newest) {
				parts[i], newest = name, fi.ModTime()
			}
		}
		return strings.Join(parts, sep)
	}
	return p
}

// SavesDir is game id's saves folder for the install pin names (the selected install when empty).
func SavesDir(home string, s settings.Settings, id, pin string) (string, error) {
	return PathFor(home, s, id, pin, PathSaves)
}

// StartupPreferencesPath is the file holding game id's startup preferences.
func StartupPreferencesPath(home string, s settings.Settings, id, pin string) (string, error) {
	return PathFor(home, s, id, pin, PathStartupPreferences)
}

// SaveLayout is how game id's saves sit in its saves folder dir: the catalog's file patterns and companions, or a
// folder per save when it names none.
func SaveLayout(id, dir string) saves.Layout {
	g, _ := catalogGame(id)
	return saves.Layout{Dir: dir, Files: g.SaveFiles, Companions: g.SaveCompanions, CompanionPaths: g.SaveCompanionPaths, Copies: g.SaveCopies, CopyFolders: g.SaveCopyFolders, Linked: g.SaveLinked, UserFolders: g.SaveUserFolders}
}

// FindSourceSites are the sites Mortar can look up a source-less mod on for the game: Nexus where the Stardew mod
// dataset lists mods by id, Thunderstore where a BepInEx plugin's GUID can match a package installed elsewhere, and the
// Vintage Story Mod DB, which answers for the mod id in modinfo.json.
func FindSourceSites(g components.GameInfo) []string {
	var out []string
	if g.HasMetadata(metadata.StardewDataset) {
		out = append(out, "nexus")
	}
	if len(g.Loaders) > 0 && g.Loaders[0].ID == "bepinex5" && slices.ContainsFunc(g.Sources, func(s components.GameSource) bool { return s.ID == "thunderstore" }) {
		out = append(out, "thunderstore")
	}
	if slices.ContainsFunc(g.Sources, func(s components.GameSource) bool { return s.ID == "vsmoddb" }) {
		out = append(out, "vsmoddb")
	}
	return out
}
