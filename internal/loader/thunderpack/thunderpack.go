// Package thunderpack is the driver of a mod loader that ships as a Thunderstore package and keeps its files in each
// profile: the profile holds the loader and the mods, the game is started on the profile through a command-line flag,
// and only the files that must sit beside the executable (a proxy library, Northstar's launcher) are placed in the
// game folder for a launch and taken out after it. Each loader is one Spec of it.
package thunderpack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
	"github.com/Rethunk-Tech/mortar/internal/wineoverride"
)

// Spec is what differs between the loaders. All paths are slash paths.
type Spec struct {
	LoaderID string
	// Name is the loader's name in messages.
	Name string
	// Root is the folder of the Thunderstore package that holds the loader's files, "" for the package's top.
	Root string
	// Placed says whether the pack file at rel (below Root, after Move) goes into the game folder for a launch.
	Placed func(rel string) bool
	// Replace are patterns of pack folders that an update deletes from the profile before the new ones are copied, so a
	// file the new release no longer ships does not stay.
	Replace []string
	// Seed are patterns of pack files copied only when the profile has none, so a player's edit survives an update.
	Seed []string
	// Move renames a top-level pack entry on its way into the profile.
	Move map[string]string
	// Map, when set, replaces Move: it is where the pack file rel goes in the profile, "" for a file the install drops.
	Map func(rel string) string
	// Dirs are folders an install creates in the profile when the pack has none, for a loader that fails without them.
	Dirs []string
	// Files are the profile's top-level paths an install lays out.
	Files []string
	// Args are the flags that start the game on the profile in dir; proton asks for paths in Wine's form.
	Args func(dir string, proton bool) []string
	// VanillaArgs start the game with the loader loaded and no mods.
	VanillaArgs []string
	// Overrides are the DLLs Wine must load natively, from the files placed.
	Overrides func(placed []string) []string
	// ProxyDLL is the one proxy library every release uses, for the Proton prefix check; "" when it differs by game.
	ProxyDLL string
	// Exe is the launcher, below the game folder, that starts in place of the game's own executable.
	Exe string
	// ImportDirs and ConfigFolders are folders of the profile: where an imported pack's loose files go and where mods
	// keep settings.
	ImportDirs, ConfigFolders []string
	// Log is the loader's log; nil for a loader that keeps none Mortar reads.
	Log *LogSpec
	// Route is where each file of package pkg goes in the profile; nil means the catalog's routes.
	Route func(files []string, pkg string) map[string]string
}

// Loader is one Spec as a registered loader. The zero fields are the real network; tests point them elsewhere.
type Loader struct {
	Spec
	// Index is the Thunderstore driver the pack is looked up in; nil means the registered one.
	Index *thunderstore.Driver
	// HTTP downloads the pack.
	HTTP *http.Client
}

type marker struct {
	Version string `json:"version"`
	// Placed are the profile paths the install put in place for the game folder.
	Placed []string `json:"placed"`
}

func (l Loader) markerFile() string { return ".mortar-" + l.LoaderID + ".json" }

func (l Loader) readMarker(dir string) marker {
	var m marker
	if b, err := fsx.ReadFile(filepath.Join(dir, l.markerFile())); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func (l Loader) ID() string { return l.LoaderID }

func (Loader) Formats() []string { return nil }

// ProfileFiles are what an install lays out in each profile's folder, with the marker that records the version.
func (l Loader) ProfileFiles() []string { return append(slices.Clone(l.Files), l.markerFile()) }

// ImportRoots are the folders an imported profile pack may carry loose.
func (l Loader) ImportRoots() []string { return l.ImportDirs }

// ConfigDirs is where the loader's mods keep their settings.
func (l Loader) ConfigDirs() []string { return l.ConfigFolders }

// OverrideDLL is the proxy library of every release, for the Proton prefix check.
func (l Loader) OverrideDLL(string) string { return l.ProxyDLL }

// VersionScheme is Thunderstore's strict d.d.d.
func (Loader) VersionScheme() string { return deps.SemverStrict }

// Status is installed when the profile holds the marker and every file it placed.
func (l Loader) Status(t loader.Target) (loader.Status, error) {
	m := l.readMarker(t.ProfileDir)
	if m.Version == "" || len(m.Placed) == 0 {
		return loader.Status{}, nil
	}
	for _, f := range m.Placed {
		if !fsx.IsFile(filepath.Join(t.ProfileDir, filepath.FromSlash(f))) {
			return loader.Status{}, nil
		}
	}
	return loader.Status{Installed: true, Version: m.Version, PerProfile: true}, nil
}

// Install lays the release's loader files out in the profile and records its version and the files to place.
func (l Loader) Install(_ context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	if err := os.MkdirAll(t.ProfileDir, 0o750); err != nil {
		return "", err
	}
	// Beside the profile, so the moves stay on one file system.
	tmp, err := os.MkdirTemp(filepath.Dir(t.ProfileDir), ".thunderpack-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = fsx.RemoveAll(tmp) }()
	if err := archive.Extract(pkg.Archive, tmp); err != nil {
		return "", err
	}
	src := filepath.Join(tmp, filepath.FromSlash(l.Root))
	files, err := packFiles(src)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("the %s package holds no %q folder with files in it", l.Name, l.Root)
	}
	// An earlier release's placed files may have other names (Lovely's proxy was version.dll before winmm.dll).
	for _, f := range l.readMarker(t.ProfileDir).Placed {
		if err := fsx.Remove(filepath.Join(t.ProfileDir, filepath.FromSlash(f))); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	for _, pat := range l.Replace {
		matches, err := filepath.Glob(filepath.Join(src, filepath.FromSlash(pat)))
		if err != nil {
			return "", err
		}
		for _, m := range matches {
			rel, err := filepath.Rel(src, m)
			if err != nil {
				return "", err
			}
			dst, err := l.localDest(filepath.ToSlash(rel))
			if err != nil {
				return "", err
			}
			if err := fsx.RemoveAll(filepath.Join(t.ProfileDir, filepath.FromSlash(dst))); err != nil {
				return "", err
			}
		}
	}
	var placed []string
	for _, rel := range files {
		dst, err := l.localDest(rel)
		if err != nil {
			return "", err
		}
		if dst == "" {
			continue
		}
		to := filepath.Join(t.ProfileDir, filepath.FromSlash(dst))
		if l.seeded(rel) && fsx.IsFile(to) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
			return "", err
		}
		if err := datadir.CopyFile(filepath.Join(src, filepath.FromSlash(rel)), to); err != nil {
			return "", err
		}
		if l.Placed != nil && l.Placed(rel) {
			placed = append(placed, dst)
		}
	}
	for _, d := range l.Dirs {
		if err := os.MkdirAll(filepath.Join(t.ProfileDir, filepath.FromSlash(d)), 0o750); err != nil {
			return "", err
		}
	}
	if progress != nil {
		progress(loader.StepFiles)
	}
	b, err := json.Marshal(marker{Version: pkg.Version, Placed: placed})
	if err != nil {
		return "", err
	}
	if err := datadir.WriteFile(filepath.Join(t.ProfileDir, l.markerFile()), b, 0o600); err != nil {
		return "", err
	}
	return pkg.Version, nil
}

// packMeta are the files Thunderstore asks for at a package's top, which never reach a profile.
var packMeta = []string{"manifest.json", "icon.png", "readme.md", "changelog.md", "license", "licence", "license.md"}

// packFiles lists the regular files below dir as slash paths, without the package's own metadata.
func packFiles(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !strings.Contains(rel, "/") && slices.Contains(packMeta, strings.ToLower(rel)) {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	return out, err
}

// dest is where the pack file rel goes in the profile.
func (l Loader) dest(rel string) string {
	if l.Map != nil {
		return l.Map(rel)
	}
	for from, to := range l.Move {
		if rel == from || strings.HasPrefix(rel, from+"/") {
			return to + strings.TrimPrefix(rel, from)
		}
	}
	return rel
}

// localDest is dest(rel), refused when it would leave the profile.
func (l Loader) localDest(rel string) (string, error) {
	dst := l.dest(rel)
	if dst != "" && !filepath.IsLocal(filepath.FromSlash(dst)) {
		return "", fmt.Errorf("the %s package places %q outside the profile", l.Name, rel)
	}
	return dst, nil
}

func (l Loader) seeded(rel string) bool {
	return slices.ContainsFunc(l.Seed, func(pat string) bool { ok, _ := path.Match(pat, rel); return ok })
}

func hostPath(dir string, proton bool) string {
	p, err := filepath.Abs(dir)
	if err != nil {
		p = dir
	}
	if proton {
		return wineoverride.ZPath(p)
	}
	return p
}

// Contribute starts the game on the profile and declares the files that sit beside its executable. Only Windows builds
// are started: they run under Proton or on Windows, where the proxy library is one the prefix loads natively.
func (l Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	if p.Platform == "linux" {
		return errors.New(l.Name + " starts only a game's Windows build, which runs under Proton")
	}
	m := l.readMarker(p.Dir)
	if len(m.Placed) == 0 {
		return errors.New(l.Name + " is not installed in this profile")
	}
	proton := p.Runtime == "proton" || p.Runtime == "mortar-proton"
	info, _ := components.Game(p.Game)
	placeDir := info.LoaderPlaceDir(l.LoaderID)
	if l.Exe != "" {
		// Steam's Proton starts the game through steam -applaunch, which cannot be told to run another executable.
		if p.Runtime == "proton" {
			return errors.New(l.Name + " starts through its own launcher, which Mortar can run directly on Windows or in its own Proton, not through Steam's")
		}
		if err := plan.OverrideExe(l.ID(), filepath.Join(p.InstallDir, filepath.FromSlash(path.Join(info.MarkerDir, l.Exe)))); err != nil {
			return err
		}
	}
	plan.AddArgs(l.Args(p.Dir, proton)...)
	for _, f := range m.Placed {
		plan.AddFile(launchplan.PlanFile{Src: filepath.Join(p.Dir, filepath.FromSlash(f)), Dst: filepath.FromSlash(path.Join(placeDir, f))})
	}
	if proton && l.Overrides != nil {
		for _, dll := range l.Overrides(m.Placed) {
			plan.RequireRuntime(launchplan.RuntimeReq{Kind: "dll-override", Key: dll, Value: "native,builtin"})
		}
	}
	return nil
}

// Vanilla starts the game with the loader but none of the mods; whatever the loader left in the game folder is told to
// stand down.
func (l Loader) Vanilla(_ context.Context, plan *launchplan.Plan, _ loader.Target) error {
	plan.AddArgs(l.VanillaArgs...)
	return nil
}

// Owns is true for a game process started on this profile's folder, in either the host or the Wine form.
func (Loader) Owns(p loader.Process, prof loader.ProfileView) bool {
	host, wine := hostPath(prof.Dir, false), hostPath(prof.Dir, true)
	return slices.ContainsFunc(p.Args, func(a string) bool { return strings.Contains(a, host) || strings.Contains(a, wine) })
}

// ModArchive reports a Thunderstore-shaped archive: manifest.json and icon.png at its root.
func (Loader) ModArchive(names []string) bool {
	has := func(want string) bool {
		return slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, want) })
	}
	return has("manifest.json") && has("icon.png")
}

// topDLLs are the placed files that are libraries at the pack's top, named without their extension.
func topDLLs(placed []string) []string {
	var out []string
	for _, f := range placed {
		if !strings.Contains(f, "/") && strings.EqualFold(path.Ext(f), ".dll") {
			out = append(out, strings.TrimSuffix(f, path.Ext(f)))
		}
	}
	return out
}
