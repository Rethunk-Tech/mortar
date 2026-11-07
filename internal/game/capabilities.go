package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
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
	info, ok := components.Game(id)
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
	info, _ := components.Game(id)
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
	g, ok := components.Game(id)
	_, has := g.Paths[role]
	return ok && has
}

// HasSaves reports whether game id has a save folder.
func HasSaves(id string) bool { return HasPath(id, PathSaves) }

// HasStartupSettings reports whether game id reads startup preferences from a file outside the install.
func HasStartupSettings(id string) bool { return HasPath(id, PathStartupPreferences) }

// PathFor resolves game id's path for role in the install pin names (the selected install when empty).
func PathFor(home string, s settings.Settings, id, pin, role string) (string, error) {
	g, ok := components.Game(id)
	t, has := g.Paths[role]
	if !ok || !has {
		return "", usererr.Wrap(usererr.NotFound, fmt.Errorf("game %q has no %s path", id, role))
	}
	return PathIn(home, s, id, pin, t)
}

// PathIn resolves the path template t for game id's install pin names (the selected install when empty), inside its
// Wine or Proton prefix when it runs in one.
func PathIn(home string, s settings.Settings, id, pin string, t components.PathTemplate) (string, error) {
	g, _ := components.Game(id)
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
	g, _ := components.Game(id)
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
	if len(g.Loaders) > 0 && bepinex5.Family(g.Loaders[0].ID) && slices.ContainsFunc(g.Sources, func(s components.GameSource) bool { return s.ID == "thunderstore" }) {
		out = append(out, "thunderstore")
	}
	if slices.ContainsFunc(g.Sources, func(s components.GameSource) bool { return s.ID == "vsmoddb" }) {
		out = append(out, "vsmoddb")
	}
	return out
}

// dataRoles are the catalog path roles a game writes while it runs, apart from mods: its saves, logs, screenshots
// and startup preferences. Roles that name a file stand for the folder holding it.
var dataRoles = map[string]bool{
	PathSaves: true, "errorLogs": true, PathStartupPreferences: true, PathScreenshots: true,
	PathLog: true, PathCrashLog: true, "unityLog": true, "userData": true,
}

var dataFileRoles = map[string]bool{PathStartupPreferences: true, PathLog: true, PathCrashLog: true, "unityLog": true}

// DataDirs are the folders a native game of the pin writes outside its install, each role's folder merged into the
// outermost one (Stardew's saves, error logs and startup preferences all sit under one config folder). Roles the
// catalog has no path for on this platform are skipped.
func DataDirs(home string, s settings.Settings, id, pin string) []string {
	g, ok := components.Game(id)
	if !ok {
		return nil
	}
	var all []string
	if u := unityDataDir(home, s, id, pin); u != "" {
		all = append(all, u)
	}
	for role, t := range g.Paths {
		if !dataRoles[role] {
			continue
		}
		p, err := PathIn(home, s, id, pin, t)
		if err != nil || p == "" {
			continue
		}
		if dataFileRoles[role] {
			p = filepath.Dir(p)
		}
		all = append(all, filepath.Clean(p))
	}
	slices.Sort(all)
	var out []string
	for _, p := range all {
		if n := len(out); n > 0 && (p == out[n-1] || strings.HasPrefix(p, out[n-1]+string(filepath.Separator))) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// unityDataDir is the folder a native Unity player keeps its saves and Player.log in, ~/.config/unity3d/<company>/<product>.
// The build records both names in <exe>_Data/app.info, so a game needs no catalog path for it. Empty when the install
// is not a native Unity build or a name could step out of the folder.
func unityDataDir(home string, s settings.Settings, id, pin string) string {
	in, err := ResolveInstall(home, s, id, pin)
	if err != nil || in.Platform != "linux" {
		return ""
	}
	company, product, ok := unityNames(in.Dir)
	if !ok {
		return ""
	}
	p, err := PathIn(home, s, id, pin, components.PathTemplate{Linux: "{xdgConfig}/unity3d/" + company + "/" + product})
	if err != nil {
		return ""
	}
	return p
}

// unityNames reads the company and product a Unity build records in <exe>_Data/app.info of dir.
func unityNames(dir string) (company, product string, ok bool) {
	infos, _ := filepath.Glob(filepath.Join(dir, "*_Data", "app.info"))
	if len(infos) == 0 {
		return "", "", false
	}
	b, err := safefs.ReadFile(infos[0])
	if err != nil {
		return "", "", false
	}
	names := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(b), "\r", "")), "\n")
	if len(names) < 2 {
		return "", "", false
	}
	for _, n := range names[:2] {
		if n == "" || n == "." || n == ".." || strings.ContainsAny(n, "/\\") {
			return "", "", false
		}
	}
	return names[0], names[1], true
}
