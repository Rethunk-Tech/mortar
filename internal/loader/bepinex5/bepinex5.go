// Package bepinex5 holds the mechanics of the BepInEx 5 mod loader that do not depend on any game or profile model: laying out the
// BepInExPack, the Doorstop proxy files and launch flags, the Wine DLL override a Proton prefix needs, and where a Thunderstore
// package's files go.
package bepinex5

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/wineoverride"
)

const (
	doorstopFile    = ".doorstop_version"
	defaultDoorstop = 3
)

// entries are the assemblies Doorstop can start BepInEx through, relative to the profile root, which BepInEx takes
// its root from: BepInEx 5's preloader, BepInEx 6's IL2CPP entry, which runs on the .NET runtime its pack ships in
// dotnet/, BepInEx 6's Mono preloader (Slip & Skid), and the IL2CPP entry of the 6.0.0-be.577 build that Crab Game's
// pack runs on the Mono runtime it ships in mono/.
var entries = []string{
	filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.Unity.IL2CPP.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.Unity.Mono.Preloader.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.IL2CPP.dll"),
}

// packRuntimeDirs are the folders a pack lays out beside BepInEx for the runtime or the core libraries Doorstop loads
// the entry assembly with: the .NET runtime of an IL2CPP pack, the Mono runtime of a legacy one, and the unstripped
// core libraries of a Mono one.
var packRuntimeDirs = []string{corlibDir, "mono", "unstripped_corlib"}

// entry is the entry assembly of the pack installed in profileRoot, relative to it.
func entry(profileRoot string) string {
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(profileRoot, e)); err == nil {
			return e
		}
	}
	return entries[0]
}

// The .NET runtime of an IL2CPP pack, relative to the profile root: Doorstop 4 reads the corlib folder and coreclr
// from its [Il2Cpp] settings or its CLR flags.
var (
	corlibDir = "dotnet"
	coreclr   = filepath.Join("dotnet", "coreclr.dll")
)

func hasCLR(profileRoot string) bool {
	_, err := os.Stat(filepath.Join(profileRoot, coreclr))
	return err == nil
}

// Installed is what InstallPack found in the pack.
type Installed struct {
	// Version is the pack's version (the Thunderstore manifest's version_number, which tracks BepInEx's own).
	Version string
	// Doorstop is the major version of the Doorstop proxy, from .doorstop_version; 3 when the pack has none.
	Doorstop int
}

// InstallPack lays the pack zip's root folder (BepInExPack/, or a community build's own name such as
// BepInExPack_Valheim/: whichever holds an entry assembly, or the zip itself for a pack with BepInEx/ at its top, as
// bbepis-BepInEx_Rogue_Tower) out into profileRoot, replacing files already there.
func InstallPack(zipPath, profileRoot string) (Installed, error) {
	if err := os.MkdirAll(profileRoot, 0o750); err != nil {
		return Installed{}, err
	}
	// Beside the profile, so the final moves stay on one file system.
	tmp, err := os.MkdirTemp(filepath.Dir(profileRoot), ".bepinex-pack-*")
	if err != nil {
		return Installed{}, err
	}
	defer func() { _ = fsx.RemoveAll(tmp) }()
	if err := archive.Extract(zipPath, tmp); err != nil {
		return Installed{}, err
	}
	src, err := packRoot(tmp)
	if err != nil {
		return Installed{}, err
	}
	var manifest Manifest
	if b, err := fsx.ReadFile(filepath.Join(tmp, "manifest.json")); err == nil {
		manifest, _ = ParseManifest(b)
	}
	if err := clearPack(profileRoot); err != nil {
		return Installed{}, err
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, p)
		if src == tmp && thunderstoreMeta[rel] {
			return nil
		}
		dst := filepath.Join(profileRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o750)
		}
		// A config file the profile already has holds the player's settings; BepInEx adds any new keys itself.
		if filepath.Dir(rel) == filepath.Join("BepInEx", "config") {
			if _, err := os.Stat(dst); err == nil {
				return nil
			}
		}
		return fsx.Rename(p, dst)
	})
	if err != nil {
		return Installed{}, err
	}
	return Installed{Version: manifest.Version, Doorstop: doorstopMajor(profileRoot)}, nil
}

// clearPack removes the Doorstop files and BepInEx core a previous pack laid out, so a pack installed over another
// keeps none of its files: an older pack has no .doorstop_version, and a newer one's left behind would make Mortar
// pass Doorstop 4 flags to a Doorstop 3 proxy, which then never starts BepInEx. A package's own core files sit in
// folders below core and stay.
func clearPack(profileRoot string) error {
	for _, name := range append(append([]string{doorstopFile, "doorstop_config.ini", "doorstop_libs"}, components.DoorstopProxies...), packRuntimeDirs...) {
		if err := fsx.RemoveAll(filepath.Join(profileRoot, name)); err != nil {
			return err
		}
	}
	core := filepath.Join(profileRoot, "BepInEx", "core")
	ents, err := os.ReadDir(core)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	for _, e := range ents {
		if !e.IsDir() {
			if err := fsx.RemoveAll(filepath.Join(core, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// thunderstoreMeta are the files Thunderstore requires at a package's top, kept out of the profile when the pack's
// BepInEx files share that top.
var thunderstoreMeta = map[string]bool{"manifest.json": true, "README.md": true, "CHANGELOG.md": true, "icon.png": true}

// packRoot is the folder of the extracted pack that holds a BepInEx entry assembly: a top-level folder, or dir itself.
func packRoot(dir string) (string, error) {
	for _, entry := range entries {
		if _, err := os.Stat(filepath.Join(dir, entry)); err == nil {
			return dir, nil
		}
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		for _, entry := range entries {
			if _, err := os.Stat(filepath.Join(dir, e.Name(), entry)); err == nil {
				return filepath.Join(dir, e.Name()), nil
			}
		}
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = filepath.ToSlash(e)
	}
	return "", errors.New("the zip has no folder holding " + strings.Join(names, " or "))
}

func doorstopMajor(profileRoot string) int {
	b, err := fsx.ReadFile(filepath.Join(profileRoot, doorstopFile))
	if err != nil {
		return defaultDoorstop
	}
	major, _, _ := strings.Cut(strings.TrimSpace(string(b)), ".")
	if n, err := strconv.Atoi(major); err == nil && n > 0 {
		return n
	}
	return defaultDoorstop
}

// proxyDLL is the Doorstop proxy the pack in profileRoot ships; before it holds a pack, the one the game's catalog entry
// names.
func proxyDLL(profileRoot, gameID string) string {
	for _, name := range components.DoorstopProxies {
		if fsx.IsFile(filepath.Join(profileRoot, name)) {
			return name
		}
	}
	info, _ := components.Game(gameID)
	return info.DoorstopProxy()
}

// DoorstopFiles lists, relative to profileRoot and with forward slashes, the proxy files that must sit next to the game
// executable: the pack's Doorstop proxy and doorstop_config.ini. Absent ones are left out. Everything else of the pack
// stays in the profile.
func DoorstopFiles(profileRoot string) []string {
	var files []string
	for _, name := range []string{doorstopConfig, proxyDLL(profileRoot, "")} {
		if _, err := os.Stat(filepath.Join(profileRoot, name)); err == nil {
			files = append(files, name)
		}
	}
	return files
}

const (
	doorstopConfig = "doorstop_config.ini"
	// relayConfig is the profile's Doorstop config with its target made absolute, the copy the game folder gets.
	relayConfig = ".mortar-doorstop_config.ini"
)

// absoluteDoorstopConfig writes the profile's Doorstop config with its target assembly set to the profile's entry
// (added, in the spelling of Doorstop doorstopMajor, when the config names none) and an IL2CPP pack's runtime paths
// made absolute, and returns the copy's path: the game folder's config then loads the profile's BepInEx even when a
// start cannot pass Doorstop's flags, as the Epic Games Launcher's link cannot. Doorstop reads relative paths from
// the game folder, where the profile's files are not.
func absoluteDoorstopConfig(profileRoot string, doorstopMajor int, proton bool) (string, error) {
	b, err := fsx.ReadFile(filepath.Join(profileRoot, doorstopConfig))
	if err != nil {
		return "", err
	}
	target := targetPath(profileRoot, proton)
	values := map[string]string{"targetAssembly": target, "target_assembly": target}
	if hasCLR(profileRoot) {
		values["corlib_dir"] = hostPath(profileRoot, corlibDir, proton)
		values["coreclr_path"] = hostPath(profileRoot, coreclr, proton)
	}
	lines := strings.Split(string(b), "\n")
	found := false
	for i, line := range lines {
		key, val, ok := strings.Cut(line, "=")
		k := strings.TrimSpace(key)
		v, known := values[k]
		if ok && !known && slices.Contains(runtimePathKeys, k) {
			v, known = profilePath(profileRoot, strings.TrimSpace(val), proton)
		}
		if !ok || !known {
			continue
		}
		found = found || v == target
		lines[i] = k + "=" + v
		if strings.HasSuffix(line, "\r") {
			lines[i] += "\r"
		}
	}
	if !found {
		key := "targetAssembly"
		if doorstopMajor >= 4 {
			key = "target_assembly"
		}
		lines = append(lines, key+"="+target)
	}
	out := filepath.Join(profileRoot, relayConfig)
	return out, fsx.WriteFile(out, []byte(strings.Join(lines, "\n")), 0o600)
}

// runtimePathKeys are the Doorstop settings (3 and 4) that name a pack's runtime folder or file relative to the game
// folder.
var runtimePathKeys = []string{"runtimeLib", "configDir", "corlibDir", "dll_search_path_override", "dllSearchPathOverride"}

// profilePath is the absolute form of a setting's relative value when it names something in the profile.
func profilePath(profileRoot, value string, proton bool) (string, bool) {
	rel := filepath.FromSlash(strings.ReplaceAll(value, `\`, "/"))
	if rel == "" || filepath.IsAbs(rel) {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(profileRoot, rel)); err != nil {
		return "", false
	}
	return hostPath(profileRoot, rel, proton), true
}

func targetPath(profileRoot string, proton bool) string {
	return hostPath(profileRoot, entry(profileRoot), proton)
}

// hostPath is rel under profileRoot as an absolute path, in the Wine Z: form for proton.
func hostPath(profileRoot, rel string, proton bool) string {
	p, err := filepath.Abs(filepath.Join(profileRoot, rel))
	if err != nil {
		p = filepath.Join(profileRoot, rel)
	}
	if proton {
		// Wine maps the host's root to Z:.
		return wineoverride.ZPath(p)
	}
	return p
}

// LaunchArgs are the Doorstop flags that start the game with BepInEx rooted in profileRoot, with an IL2CPP pack's
// .NET runtime. Doorstop 3 and 4 spell the flags differently; proton gives the paths in the Wine Z: form.
func LaunchArgs(profileRoot string, doorstopMajor int, proton bool) []string {
	target := targetPath(profileRoot, proton)
	if doorstopMajor >= 4 {
		args := []string{"--doorstop-enabled", "true", "--doorstop-target-assembly", target}
		if hasCLR(profileRoot) {
			args = append(args,
				"--doorstop-clr-corlib-dir", hostPath(profileRoot, corlibDir, proton),
				"--doorstop-clr-runtime-coreclr-path", hostPath(profileRoot, coreclr, proton))
		}
		return args
	}
	return []string{"--doorstop-enable", "true", "--doorstop-target", target}
}

// VanillaArgs start the game with Doorstop switched off, even while its files sit in the game folder. A vanilla start
// has no profile to read the Doorstop version from, and the proxy left in the folder may be either, so both
// spellings are passed; each proxy ignores the other's.
func VanillaArgs() []string {
	return []string{"--doorstop-enable", "false", "--doorstop-enabled", "false"}
}

// Route is where a file of Thunderstore package pkg (Namespace-Name) goes in the profile root, from its path relPath
// inside the package zip, or "" when the path is unsafe. It follows r2modman's BepInEx rules
// (r2modmanPlus src/installers/InstallRulePluginInstaller.ts, buildInstallForRuleSubtype), which mod authors build
// and test against: the first folder on the path named plugins, patchers, monomod, core or config, at any depth, is
// that rule's folder and keeps the structure below it (plugins, patchers, monomod and core in a folder named for the
// package, config flat). A file under no such folder is placed by its name alone, a .mm.dll in monomod and anything
// else in plugins (the package's own manifest.json and icon.png too, which in-game mod menus read), so the folders
// above it are flattened away: packages such as MirageCore ship FSharp.Core/FSharp.Core.dll
// and load it from beside their plugin. The game's routes (components.LoaderRoute) add folders of the same kind, each
// matched by its last segment at any depth, and extensions that send a file there wherever it sits.
func Route(relPath, pkg string, routes []components.LoaderRoute) string {
	if pkg == "" || strings.ContainsAny(pkg, `/\:`) || pkg == "." || pkg == ".." {
		return ""
	}
	p := path.Clean(strings.ReplaceAll(relPath, `\`, "/"))
	if p == "." || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, ":") {
		return ""
	}
	segs := strings.Split(p, "/")
	for i, dir := range segs[:len(segs)-1] {
		rest := path.Join(segs[i+1:]...)
		if r, ok := components.RouteByFolder(routes, dir); ok {
			return r.Place(pkg, rest, p)
		}
		switch strings.ToLower(dir) {
		case "config":
			return path.Join("BepInEx", "config", rest)
		case "plugins", "patchers", "monomod", "core":
			return path.Join("BepInEx", strings.ToLower(dir), pkg, rest)
		}
	}
	name := segs[len(segs)-1]
	if r, ok := components.RouteByExtension(routes, name); ok {
		return r.Place(pkg, name, p)
	}
	if strings.HasSuffix(strings.ToLower(name), ".mm.dll") {
		return path.Join("BepInEx", "monomod", pkg, name)
	}
	return path.Join("BepInEx", "plugins", pkg, name)
}

// Manifest is the part of a Thunderstore package's manifest.json that Mortar reads.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version_number"`
	Namespace    string   `json:"namespace"`
	Author       string   `json:"author"`
	Dependencies []string `json:"dependencies"`
}

// Needs is the packages the manifest depends on, each at least the version it names. The BepInEx pack is left out:
// the loader install supplies it, so no profile entry ever stands for it.
func (m Manifest) Needs() []manifest.Dependency {
	var out []manifest.Dependency
	for _, s := range m.Dependencies {
		d, err := deps.Thunderstore(s)
		if err != nil || components.IsLoaderPackage(strings.TrimPrefix(d.Target.Package, "thunderstore:")) {
			continue
		}
		out = append(out, manifest.NewDependency(mod.ID(d.Target.Package), d.Constraint, true))
	}
	return out
}

// ParseManifest reads a Thunderstore manifest.json. Thunderstore accepts manifests saved with a UTF-8 BOM, and many
// packages ship one, but encoding/json refuses it.
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	err := json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &m)
	return m, err
}
