package problems

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
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
	Line       int    `json:"line"`
}

// loaderFailures runs the loader's log analyzers over its log in the profile and the Unity player log text, and attributes each finding to an enabled
// package. A loader without a log, a missing log or a log without findings yields none.
func loaderFailures(l loader.Loader, p loader.ProfileView, player string, owners func() map[string]framework.Mod) []LoadFailure {
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
	if len(found) == 0 {
		return nil
	}
	byName := owners()
	out := make([]LoadFailure, 0, len(found))
	for _, f := range found {
		lf := LoadFailure{Plugin: f.Plugin, Kind: f.Kind, Message: f.Message, Dependency: f.Dependency, Line: f.Line}
		if owner, ok := byName[strings.ToLower(f.Plugin)]; ok {
			lf.Key, lf.ID, lf.Name = owner.Key, owner.ModID(), owner.Name
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

// pluginOwners maps every way a log names a plugin (its GUID, its name, or "name version") to the enabled package
// whose plugin DLLs declare it. A package is the identity; the GUIDs are read from its files, since a package may hold
// several plugins or none and can change a GUID between versions.
func pluginOwners(mods []framework.Mod) map[string]framework.Mod {
	owners := map[string]framework.Mod{}
	for _, m := range mods {
		if !m.Enabled || m.Folder == "" {
			continue
		}
		_ = filepath.WalkDir(m.Folder, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".dll") {
				return nil
			}
			plugins, _ := dotnet.Plugins(path)
			for _, pl := range plugins {
				for _, name := range []string{pl.GUID, pl.Name, pl.Name + " " + pl.Version} {
					owners[strings.ToLower(name)] = m
				}
			}
			return nil
		})
	}
	return owners
}
