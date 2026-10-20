package problems

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/loader/gamelog"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// LoadFailure is a plugin the loader's log says failed to load or ran into errors. Key and Name are the installed
// package the plugin belongs to (ID is its mod id); both are empty, and Plugin is shown as the log wrote it, when no package declares it.
type LoadFailure struct {
	Key     string `json:"key"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Plugin  string `json:"plugin"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	// Dependency is the plugin the failing one lacks, when the log names it.
	Dependency string `json:"dependency,omitempty"`
	// Path is a skipped older copy no installed package owns, relative to the profile folder with slashes: a file
	// left in the plugins folder that RemoveSkippedPlugin can delete.
	Path string `json:"path,omitempty"`
	Line int    `json:"line"`
	// Ran marks an error the plugin hit after it loaded: it ran, so this is not a load failure. A mod's errors of this
	// kind are one row, the first with Errors counting them all.
	Ran    bool `json:"ran,omitempty"`
	Errors int  `json:"errors,omitempty"`
	// Thrown is the method a ran error's stack was thrown in, "Type.Method", and PatchedBy the other enabled mods whose
	// Harmony patches target a method on that stack: the suspects when the code that threw is the game's.
	Thrown    string   `json:"thrown,omitempty"`
	PatchedBy []string `json:"patchedBy,omitempty"`
}

// pluginsDir is where BepInEx loads plugins from, below the profile folder.
const pluginsDir = "BepInEx/plugins"

// loaderFailures runs the loader's log analyzers over its log in the profile and the Unity player log text, and attributes each finding to an enabled
// package. A loader without a log, a missing log or a log without findings yields none.
func loaderFailures(l loader.Loader, p loader.ProfileView, player string, owners func() map[string]framework.Mod, patchers func(methods []string) []string) []LoadFailure {
	logs, ok := l.(loader.WithLogs)
	if !ok {
		return nil
	}
	path, err := logs.Path(p)
	if err != nil {
		return nil
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return nil
	}
	var found []loader.Finding
	for _, a := range logs.Analyzers() {
		found = append(found, a.Analyze(loader.Logs{Loader: string(raw), Player: player})...)
	}
	logLines := map[string][]string{}
	stackOf := func(f loader.Finding) []string {
		text := string(raw)
		if f.Source == "Player.log" {
			text = player
		}
		lines, ok := logLines[f.Source]
		if !ok {
			lines = strings.Split(text, "\n")
			logLines[f.Source] = lines
		}
		if f.Line <= 0 || f.Line > len(lines) {
			return nil
		}
		return dotnetstack.Methods(lines[f.Line:])
	}
	out := attribute(found, owners, func(lf *LoadFailure, f loader.Finding) { explain(lf, stackOf(f), patchers) })
	if l.ID() == bepinex5.ID {
		out = dropFixed(out, filepath.Join(p.Dir, filepath.FromSlash(pluginsDir)), owners)
	}
	for i, f := range out {
		if f.Kind != bepinex5.KindDuplicateGUID || f.Key != "" {
			continue
		}
		if abs := dotnet.PluginFile(filepath.Join(p.Dir, filepath.FromSlash(pluginsDir)), f.Plugin); abs != "" {
			rel, _ := filepath.Rel(p.Dir, abs)
			out[i].Path = filepath.ToSlash(rel)
		}
	}
	return out
}

// namedByPlugin is the finding kinds whose Plugin is the plugin's GUID, name or "name version" as BepInEx logged it,
// rather than a namespace read off a stack.
var namedByPlugin = []string{
	bepinex5.KindMissingDependency, bepinex5.KindIncompatible, bepinex5.KindIncompatiblePlugin,
	bepinex5.KindLoaderVersion, bepinex5.KindLoadException, bepinex5.KindDependencyNotLoaded,
}

// dropFixed removes the findings the profile has since fixed, which stay in the log until the next run: a missing
// dependency that an enabled package now declares, and a finding about a plugin that no plugin file in the profile
// declares any more (updated to another version, replaced, or its package left).
func dropFixed(found []LoadFailure, pluginsPath string, owners func() map[string]framework.Mod) []LoadFailure {
	if len(found) == 0 {
		return found
	}
	byName := owners()
	var declared map[string]bool
	gone := func(plugin string) bool {
		if declared == nil {
			declared = map[string]bool{}
			for _, pl := range dotnet.PluginsIn(pluginsPath) {
				for _, n := range []string{pl.GUID, pl.Name, pl.Name + " " + pl.Version} {
					declared[strings.ToLower(n)] = true
				}
			}
		}
		return !declared[strings.ToLower(plugin)]
	}
	return slices.DeleteFunc(found, func(f LoadFailure) bool {
		if f.Kind == bepinex5.KindMissingDependency && f.Dependency != "" {
			if _, ok := byName[strings.ToLower(f.Dependency)]; ok {
				return true
			}
		}
		return slices.Contains(namedByPlugin, f.Kind) && f.Plugin != "" && gone(f.Plugin)
	})
}

// RemoveSkippedPlugin deletes a DLL a LoadFailure's Path named, which BepInEx skipped for a newer copy of the same
// plugin. Only a plugin file below the profile's plugins folder qualifies.
func (s *Service) RemoveSkippedPlugin(gameID, id, path string) error {
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return err
	}
	rel := filepath.FromSlash(path)
	if !filepath.IsLocal(rel) || !strings.EqualFold(filepath.Ext(rel), ".dll") || !strings.HasPrefix(filepath.ToSlash(rel), pluginsDir+"/") {
		return fmt.Errorf("%s is not a plugin file in this profile", path)
	}
	abs := filepath.Join(dir, rel)
	if info, err := os.Lstat(abs); err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a plugin file in this profile", path)
	}
	if err := fsx.Remove(abs); err != nil {
		return err
	}
	s.ForgetCached()
	return nil
}

// attribute turns findings into load failures, naming the installed package behind each plugin the owners know.
func attribute(found []loader.Finding, owners func() map[string]framework.Mod, explainRan func(*LoadFailure, loader.Finding)) []LoadFailure {
	if len(found) == 0 {
		return nil
	}
	byName := owners()
	out := make([]LoadFailure, 0, len(found))
	ranAt := map[string]int{}
	for _, f := range found {
		lf := LoadFailure{Plugin: f.Plugin, Kind: f.Kind, Message: f.Message, Dependency: f.Dependency, Line: f.Line}
		if owner, ok := byName[strings.ToLower(f.Plugin)]; ok {
			lf.Key, lf.ID, lf.Name = owner.Key, owner.ModID(), owner.Name
		}
		if bepinex5.RanAfterLoad(f.Kind) {
			who := lf.Key
			if who == "" {
				who = strings.ToLower(f.Plugin)
			}
			if i, ok := ranAt[who]; ok {
				out[i].Errors++
				continue
			}
			ranAt[who] = len(out)
			lf.Ran, lf.Errors = true, 1
			if explainRan != nil {
				explainRan(&lf, f)
			}
		}
		out = append(out, lf)
	}
	return out
}

// gameVersionFailure is the loader's complaint that the installed game is older than the loader's component accepts.
// A loader that declares no minimum, or a game whose version is unknown, has none.
func gameVersionFailure(gameID string, l loader.Loader, env Environment) (LoadFailure, bool) {
	accepted := components.BundledAccepted(gameID, l.ID())
	minimum, ok := strings.CutPrefix(accepted, ">=")
	if !ok || env.GameVersion == "" || deps.Satisfies(env.VersionScheme, env.GameVersion, minimum) {
		return LoadFailure{}, false
	}
	return LoadFailure{
		Plugin: l.ID(), Kind: KindGameVersion,
		Message: "game version " + env.GameVersion + " is older than the loader accepts (" + accepted + ")",
	}, true
}

// KindGameVersion is the LoadFailure kind of a game version the loader does not accept.
const KindGameVersion = "game-version"

// KindRunFailed is the LoadFailure kind of a newest run that failed to start and left no log to analyse.
const KindRunFailed = "run-failed"

// startFailure is the row for a newest run that never started. Plugin carries the run id, which is what makes the row
// go away: the next recorded run replaces it.
func startFailure(runID string, summary launch.Summary) []LoadFailure {
	if summary.StartFailure == "" {
		return nil
	}
	return []LoadFailure{{Plugin: runID, Kind: KindRunFailed, Message: summary.StartFailure}}
}

// pluginOwners maps every way a log names a plugin (see dotnet.Owners) to the enabled package whose plugin DLLs
// declare it. A package is the identity; the GUIDs are read from its files, since a package may hold several plugins or
// none and can change a GUID between versions.
func pluginOwners(mods []framework.Mod) map[string]framework.Mod {
	var with []framework.Mod
	var dirs []string
	for _, m := range mods {
		if m.Enabled && m.Folder != "" {
			with, dirs = append(with, m), append(dirs, m.Folder)
		}
	}
	owners := map[string]framework.Mod{}
	for name, i := range dotnet.Owners(dirs) {
		owners[name] = with[i]
	}
	return owners
}

// gameLogFailures reads the game's own log, for a game whose mods have no loader log: Vintage Story's mod loader writes
// its failures there under the mod's id.
func gameLogFailures(text string, mods []framework.Mod) []LoadFailure {
	return attribute(gamelog.Analyze(text), func() map[string]framework.Mod { return modIDOwners(mods) }, nil)
}

// modIDOwners maps the lower-cased id and name of each enabled mod to it, which is how such a log names a mod.
func modIDOwners(mods []framework.Mod) map[string]framework.Mod {
	owners := map[string]framework.Mod{}
	for _, m := range mods {
		if m.Enabled {
			owners[strings.ToLower(m.ModID().Local())] = m
			owners[strings.ToLower(m.Name)] = m
		}
	}
	return owners
}

// explain names where a ran error was thrown and the other mods that patch a method on its stack.
func explain(lf *LoadFailure, methods []string, patchers func([]string) []string) {
	if len(methods) == 0 {
		return
	}
	lf.Thrown = strings.Replace(methods[0], "::", ".", 1)
	if patchers == nil {
		return
	}
	lf.PatchedBy = slices.DeleteFunc(patchers(methods), func(name string) bool { return name == lf.Name })
}
