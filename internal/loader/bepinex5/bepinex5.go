// Package bepinex5 holds the mechanics of the BepInEx 5 mod loader that do not depend on any game or profile model: laying out the
// BepInExPack, the Doorstop proxy files and launch flags, the Wine DLL override a Proton prefix needs, and where a Thunderstore
// package's files go.
package bepinex5

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	doorstopFile    = ".doorstop_version"
	defaultDoorstop = 3
)

// preloader is the file BepInEx takes its root from, relative to the profile root.
var preloader = filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll")

// Installed is what InstallPack found in the pack.
type Installed struct {
	// Version is the pack's version (the Thunderstore manifest's version_number, which tracks BepInEx's own).
	Version string
	// Doorstop is the major version of the Doorstop proxy, from .doorstop_version; 3 when the pack has none.
	Doorstop int
}

// InstallPack lays the pack zip's root folder (BepInExPack/, or a community build's own name such as
// BepInExPack_Valheim/: whichever holds the preloader) out into profileRoot, replacing files already there.
func InstallPack(zipPath, profileRoot string) (Installed, error) {
	if err := os.MkdirAll(profileRoot, 0o750); err != nil {
		return Installed{}, err
	}
	// Beside the profile, so the final moves stay on one file system.
	tmp, err := os.MkdirTemp(filepath.Dir(profileRoot), ".bepinex-pack-*")
	if err != nil {
		return Installed{}, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
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
	for _, name := range []string{doorstopFile, "winhttp.dll", "doorstop_config.ini", "doorstop_libs"} {
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

// packRoot is the top-level folder of the extracted pack that holds BepInEx's preloader.
func packRoot(dir string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), preloader)); err == nil && e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", errors.New("the zip has no folder holding " + filepath.ToSlash(preloader))
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

// DoorstopFiles lists, relative to profileRoot and with forward slashes, the proxy files that must sit next to the game
// executable: winhttp.dll and doorstop_config.ini. Absent ones are left out. Everything else of the pack stays in the profile.
func DoorstopFiles(profileRoot string) []string {
	var files []string
	for _, name := range []string{"doorstop_config.ini", "winhttp.dll"} {
		if _, err := os.Stat(filepath.Join(profileRoot, name)); err == nil {
			files = append(files, name)
		}
	}
	return files
}

func targetPath(profileRoot string, proton bool) string {
	p, err := filepath.Abs(filepath.Join(profileRoot, preloader))
	if err != nil {
		p = filepath.Join(profileRoot, preloader)
	}
	if proton {
		// Wine maps the host's root to Z:.
		return `Z:` + strings.ReplaceAll(filepath.ToSlash(p), "/", `\`)
	}
	return p
}

// LaunchArgs are the Doorstop flags that start the game with BepInEx rooted in profileRoot. Doorstop 3 and 4 spell the
// flags differently; proton gives the target in the Wine Z: form.
func LaunchArgs(profileRoot string, doorstopMajor int, proton bool) []string {
	target := targetPath(profileRoot, proton)
	if doorstopMajor >= 4 {
		return []string{"--doorstop-enabled", "true", "--doorstop-target-assembly", target}
	}
	return []string{"--doorstop-enable", "true", "--doorstop-target", target}
}

// VanillaArgs start the game with Doorstop switched off, even while its files sit in the game folder. A vanilla start
// has no profile to read the Doorstop version from, and the proxy left in the folder may be either, so both
// spellings are passed; each proxy ignores the other's.
func VanillaArgs() []string {
	return []string{"--doorstop-enable", "false", "--doorstop-enabled", "false"}
}

const (
	dllOverridesHeader = `[Software\\Wine\\DllOverrides]`
	winhttpLine        = `"winhttp"="native,builtin"`
	backupSuffix       = ".mortar-backup"
	// filetimeEpoch is the seconds between 1601 and 1970, for Wine's #time= lines.
	filetimeEpoch = 11644473600
)

// EnsureWinHTTPOverride makes the Proton prefix's user.reg load winhttp natively, as Doorstop needs. It is idempotent,
// keeps the file's line endings and other sections, and copies the file to user.reg.mortar-backup once before the
// first change. Run it while Wine is not running, or wineserver writes its own copy back over it.
func EnsureWinHTTPOverride(userReg string) error {
	raw, err := fsx.ReadFile(userReg)
	if err != nil {
		return err
	}
	text := string(raw)
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.Split(text, eol)
	updated, changed := withWinHTTP(lines, nowFiletime)
	if !changed {
		return nil
	}
	backup := userReg + backupSuffix
	if _, err := os.Stat(backup); errors.Is(err, fs.ErrNotExist) {
		if err := fsx.WriteFile(backup, raw, 0o600); err != nil {
			return err
		}
	}
	return datadir.WriteFile(userReg, []byte(strings.Join(updated, eol)), 0o600)
}

// withWinHTTP returns lines with the winhttp override in the DllOverrides section, adding the section when it is missing.
func withWinHTTP(lines []string, now func() (unix int64)) ([]string, bool) {
	start := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.ToLower(l), strings.ToLower(dllOverridesHeader)) })
	if start < 0 {
		unix := now()
		filetime := (unix + filetimeEpoch) * 10_000_000
		out := slices.Clone(lines)
		// A file ends with a newline, so its last element is empty and the blank line before a section already exists.
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
		return append(out, "", fmt.Sprintf("%s %d", dllOverridesHeader, unix), fmt.Sprintf("#time=%x", filetime), winhttpLine, ""), true
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(lines[end], "[") && lines[end] != "" {
		end++
	}
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(strings.ToLower(lines[i]), `"winhttp"=`) {
			if lines[i] == winhttpLine {
				return lines, false
			}
			out := slices.Clone(lines)
			out[i] = winhttpLine
			return out, true
		}
	}
	out := slices.Clone(lines[:end])
	out = append(out, winhttpLine)
	return append(out, lines[end:]...), true
}

func nowFiletime() int64 { return time.Now().Unix() }

// Route is where a file of Thunderstore package pkg (Namespace-Name) goes in the profile root, from its path relPath
// inside the package zip, or "" when the file is package metadata or unsafe. It follows r2modman's BepInEx rules
// (r2modmanPlus src/installers/InstallRulePluginInstaller.ts, buildInstallForRuleSubtype), which mod authors build
// and test against: the first folder on the path named plugins, patchers, monomod, core or config, at any depth, is
// that rule's folder and keeps the structure below it (plugins, patchers, monomod and core in a folder named for the
// package, config flat). A file under no such folder is placed by its name alone, a .mm.dll in monomod and anything
// else in plugins, so the folders above it are flattened away: packages such as MirageCore ship FSharp.Core/FSharp.Core.dll
// and load it from beside their plugin.
func Route(relPath, pkg string) string {
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
		switch strings.ToLower(dir) {
		case "config":
			return path.Join("BepInEx", "config", rest)
		case "plugins", "patchers", "monomod", "core":
			return path.Join("BepInEx", strings.ToLower(dir), pkg, rest)
		}
	}
	name := segs[len(segs)-1]
	if len(segs) == 1 {
		switch strings.ToLower(name) {
		case "manifest.json", "icon.png", "readme.md", "changelog.md":
			return ""
		}
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
