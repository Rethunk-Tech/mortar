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
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const (
	packRoot        = "BepInExPack"
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

// InstallPack lays the BepInExPack zip's BepInExPack/ folder out into profileRoot, replacing files already there.
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
	src := filepath.Join(tmp, packRoot)
	if !fsx.IsDir(src) {
		return Installed{}, fmt.Errorf("the zip has no %s folder", packRoot)
	}
	var manifest Manifest
	if b, err := fsx.ReadFile(filepath.Join(tmp, "manifest.json")); err == nil {
		manifest, _ = ParseManifest(b)
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
		return fsx.Rename(p, dst)
	})
	if err != nil {
		return Installed{}, err
	}
	return Installed{Version: manifest.Version, Doorstop: doorstopMajor(profileRoot)}, nil
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

// VanillaArgs start the game with Doorstop switched off, even while its files sit in the game folder.
func VanillaArgs(doorstopMajor int) []string {
	if doorstopMajor >= 4 {
		return []string{"--doorstop-enabled", "false"}
	}
	return []string{"--doorstop-enable", "false"}
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
	tmp := userReg + ".mortar-tmp"
	if err := fsx.WriteFile(tmp, []byte(strings.Join(updated, eol)), 0o600); err != nil {
		return err
	}
	return fsx.Rename(tmp, userReg)
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
// inside the package zip, or "" when the file is package metadata or unsafe. It follows r2modman's BepInEx rules: plugins,
// patchers, monomod and core go in a folder named for the package under BepInEx/, config is flat, and a loose DLL
// is a plugin. A BepInEx/ prefix on the path is optional.
func Route(relPath, pkg string) string {
	if pkg == "" || strings.ContainsAny(pkg, `/\:`) || pkg == "." || pkg == ".." {
		return ""
	}
	p := path.Clean(strings.ReplaceAll(relPath, `\`, "/"))
	if p == "." || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, ":") {
		return ""
	}
	if first, rest, ok := strings.Cut(p, "/"); ok && strings.EqualFold(first, "BepInEx") {
		p = rest
	}
	dir, rest, nested := strings.Cut(p, "/")
	if !nested {
		switch strings.ToLower(p) {
		case "manifest.json", "icon.png", "readme.md", "changelog.md":
			return ""
		}
		if strings.HasSuffix(strings.ToLower(p), ".mm.dll") {
			return path.Join("BepInEx", "monomod", pkg, p)
		}
		return path.Join("BepInEx", "plugins", pkg, p)
	}
	switch strings.ToLower(dir) {
	case "config":
		return path.Join("BepInEx", "config", rest)
	case "plugins", "patchers", "monomod", "core":
		return path.Join("BepInEx", strings.ToLower(dir), pkg, rest)
	}
	return path.Join("BepInEx", "plugins", pkg, p)
}

// Manifest is the part of a Thunderstore package's manifest.json that Mortar reads.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version_number"`
	Namespace    string   `json:"namespace"`
	Author       string   `json:"author"`
	Dependencies []string `json:"dependencies"`
}

// ParseManifest reads a Thunderstore manifest.json. Thunderstore accepts manifests saved with a UTF-8 BOM, and many
// packages ship one, but encoding/json refuses it.
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	err := json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &m)
	return m, err
}
