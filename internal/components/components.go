// Package components loads the signed component manifest and downloads its assets.
package components

import (
	"bytes"
	"cmp"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/github"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

const (
	// ReleasesURL lists Mortar's releases. Published releases are immutable, so every signed manifest is its own
	// components-v2-<serial> release and the newest one is the current manifest.
	ReleasesURL = "https://api.github.com/repos/Rethunk-Tech/mortar/releases?per_page=100"
	// Released clients accept any manifest whose fields they recognise and ignore the rest, so a manifest in a changed
	// schema goes under a new prefix (and cache name) that older clients never read.
	tagPrefix = "components-v2-"
	cacheName = "components-v2-manifest.json"
	day       = 24 * time.Hour

	maxManifest  = 8 << 20
	maxSignature = 1 << 20
	maxComponent = 2 << 30
)

//go:embed components.json
var bundledJSON []byte

// Source identifies the host and project that publishes a component.
type Source struct {
	Host      string `json:"host"`
	Owner     string `json:"owner,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// SourceComponent is one entry in components.source.json.
type SourceComponent struct {
	Game         string `json:"game"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	Source       Source `json:"source"`
	AssetPattern string `json:"asset"`
	Version      string `json:"version"`
	Accepted     string `json:"accepted,omitempty"`
}

// SourceFile is the committed input read by cmd/components.
type SourceFile struct {
	Components []SourceComponent `json:"components"`
	Games      []GameInfo        `json:"games"`
}

// KnownBroken is one curated entry: a mod or package that does not work in some versions of the game's mods or of the
// game itself.
type KnownBroken struct {
	// ID is the mod id or, for a Thunderstore package, its "Namespace-Name"; matched without regard to case.
	ID string `json:"id"`
	// Versions limits the entry to the affected versions of the mod: constraints separated by spaces, each one of
	// <, <=, >, >= or = followed by a version ("<2.1.0", ">=1.0 <1.5"). Empty means every version.
	Versions string `json:"versions,omitempty"`
	// Reason is the sentence the Problems row shows.
	Reason string `json:"reason"`
	// Replacement is a package ("Namespace-Name") to use instead, optional.
	Replacement string `json:"replacement,omitempty"`
}

// versionConstraint is one operator and version of KnownBroken.Versions.
var versionConstraint = regexp.MustCompile(`^(<=|>=|<|>|=)\S+$`)

// Validate checks that an entry names a mod, gives a reason and has a readable version range.
func (k KnownBroken) Validate() error {
	if strings.TrimSpace(k.ID) == "" || strings.TrimSpace(k.Reason) == "" {
		return errors.New("a known-broken entry needs an id and a reason")
	}
	for c := range strings.FieldsSeq(k.Versions) {
		if !versionConstraint.MatchString(c) {
			return fmt.Errorf("known-broken entry %q has an unreadable version range %q", k.ID, k.Versions)
		}
	}
	return nil
}

// GameInfo is one game's catalog entry: the names and ids the stores and mod sites know it by, its loaders and its mod
// sources. How a game is launched or modded stays in its Go implementation; this is only what can change without a
// Mortar release.
type GameInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Enabled is false for a game listed as coming later.
	Enabled bool `json:"enabled"`
	// ImportIDs are the names Vortex and Mod Organizer 2 give the game, so their profiles can be imported; a game
	// neither manager has an extension or plugin for has none.
	ImportIDs ImportIDs `json:"importIds,omitzero"`
	// Marker is a file every install of the game holds, at its root or one "game" folder down.
	Marker string `json:"marker"`
	// MarkerDir is the slash-separated folder below the install root that holds Marker, for a game whose executable
	// sits below the root (Game/Bin); empty when the marker is at the root or one "game" folder down.
	MarkerDir string `json:"markerDir,omitempty"`
	// LinuxMarker is the executable of the game's native Linux build where the store ships one beside the build Marker
	// names: an install holding it and not Marker is that build, started natively rather than under Proton.
	LinuxMarker string `json:"linuxMarker,omitempty"`
	// R2modmanFolder is r2modman's own folder name for the game (the Thunderstore schema's internalFolderName), the
	// parent of its profiles folder.
	R2modmanFolder string `json:"r2modmanFolder,omitempty"`
	// Metadata names the SMAPI-derived features that apply to the game: smapi-updates, smapi-compat, stardew-dataset.
	Metadata []string `json:"metadata"`
	// NewProfileSeparateSaves starts a new profile of this game with its own saves rather than the shared ones.
	NewProfileSeparateSaves bool `json:"newProfileSeparateSaves,omitempty"`
	// Paths names folders and files outside the install by role (saves, startupPreferences, options). The options
	// role is a file; the others are folders.
	Paths map[string]PathTemplate `json:"paths,omitempty"`
	// SaveFiles are patterns naming each save file in the saves folder or one folder below it, for a game that keeps a
	// save as a file; a pattern starting with "!" excludes the files it matches. Without them a save is a folder holding
	// a file of its own name, as Stardew Valley's are.
	SaveFiles []string `json:"saveFiles,omitempty"`
	// SaveCompanions are extensions of files beside a save file, sharing its stem, that belong to that save (a Valheim
	// world's .db beside its .fwl).
	SaveCompanions []string `json:"saveCompanions,omitempty"`
	// RequiredSettings are settings in a game's own files that mods need switched on; Problems reports one that is
	// switched off and never edits the file. An older Mortar ignores the field.
	RequiredSettings []RequiredSetting `json:"requiredSettings,omitempty"`
	// Caches are rebuildable cache files or folders the game rebuilds on launch, which Mortar deletes once after a
	// profile's mod set changes.
	Caches []CachePath `json:"caches,omitempty"`
	// KnownBroken is Mortar's curated list of mods and packages of this game that are known not to work; an older
	// Mortar ignores the field.
	KnownBroken []KnownBroken `json:"knownBroken,omitempty"`
	// TitleScene is the Unity scene of the game's main menu, where a BepInEx startup measurement ends; without it the
	// first scene ends it.
	TitleScene string `json:"titleScene,omitempty"`
	// Graphics offers the player a choice of graphics API for the game, each passed as launch arguments; an older
	// Mortar ignores the field.
	Graphics *Graphics `json:"graphics,omitempty"`
	// Deploy is how the profile reaches the game: redirect (the loader points the game at the profile's mods folder,
	// nothing is placed) or profile (the profile holds the loader and its mods, and the loader's install-side files are
	// placed into the install for the launch and taken back after).
	Deploy string `json:"deploy"`
	// Targets are the places the game's mod files go.
	Targets []TargetDef  `json:"targets"`
	Stores  GameStores   `json:"stores"`
	Loaders []GameLoader `json:"loaders"`
	Sources []GameSource `json:"sources"`
	// Templates are starter profiles offered when creating one: a named list of packages, each queued through the
	// normal download path.
	Templates []StarterTemplate `json:"templates,omitempty"`
}

// Graphics is a game's graphics API choices.
type Graphics struct {
	// Explanation is one paragraph, in plain words, on what the choice changes.
	Explanation string           `json:"explanation"`
	Choices     []GraphicsChoice `json:"choices"`
	// Recommended is the id of the choice to steer the player to, Reason why, and Source where that is said.
	Recommended string `json:"recommended"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
}

// GraphicsChoice is one graphics API: its launch arguments are empty for the game's own default.
type GraphicsChoice struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Args  []string `json:"args,omitempty"`
}

// Choice is the choice with id.
func (g Graphics) Choice(id string) (GraphicsChoice, bool) {
	for _, c := range g.Choices {
		if c.ID == id {
			return c, true
		}
	}
	return GraphicsChoice{}, false
}

// Validate rejects a block that cannot be offered: two or more uniquely named choices, one of them recommended with a
// reason and a source.
func (g Graphics) Validate() error {
	if g.Explanation == "" || g.Reason == "" || g.Source == "" || len(g.Choices) < 2 {
		return errors.New("graphics needs an explanation, two choices, and a recommendation with a reason and a source")
	}
	seen := map[string]bool{}
	for _, c := range g.Choices {
		if c.ID == "" || c.Label == "" || seen[c.ID] {
			return fmt.Errorf("graphics choice %q needs a unique id and a label", c.ID)
		}
		seen[c.ID] = true
	}
	if !seen[g.Recommended] {
		return fmt.Errorf("graphics recommends %q, which is not a choice", g.Recommended)
	}
	return nil
}

// RequiredSetting names one `key = value` line of the file at the game's path role Path that mods need: a file that
// holds the key with another value is reported with Message; a file or key that is absent is not.
type RequiredSetting struct {
	Path string `json:"path"`
	// Section is the ini section the key sits in; empty means the key is looked up anywhere in the file.
	Section string `json:"section,omitempty"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// CachePath names a rebuildable cache file or folder below a path role of the game, as a slash path with no "..".
type CachePath struct {
	Role string `json:"role"`
	Path string `json:"path"`
}

// StarterTemplate is a built-in starting set of mods for a game.
type StarterTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Packages    []TemplatePackage `json:"packages"`
}

// TemplatePackage names one mod by the source that serves it: a Nexus mod id, a GitHub "owner/repo", or a
// Thunderstore "Namespace-Name". Name is the label shown before it is downloaded.
type TemplatePackage struct {
	Source string `json:"source"`
	Ref    string `json:"ref"`
	Name   string `json:"name"`
}

// ImportIDs name a game to the managers whose profiles Mortar imports: Vortex's game id (its game extension's
// GAME_ID) and MO2's game plugin name (GameName).
type ImportIDs struct {
	Vortex string `json:"vortex,omitempty"`
	MO2    string `json:"mo2,omitempty"`
}

// PathTemplate is a path per platform of the game build. Tokens: {appData} {localAppData} {localLow} {documents}
// {xdgConfig} {xdgData} {home} {install}; the runtime that runs the install gives them their folders.
type PathTemplate struct {
	Windows string `json:"windows,omitempty"`
	Linux   string `json:"linux,omitempty"`
	Darwin  string `json:"darwin,omitempty"`
}

// Deploy methods.
const (
	DeployRedirect = "redirect"
	DeployProfile  = "profile"
)

// TargetDef is a content target: a named place mod files go. Root is where it lives in the profile: {profileMods}
// (the profile's mods folder), {profile} (the profile's root) or a folder below {profile}. MaxDepth caps, per
// lower-case file extension, how many folders deep a file may sit below the target.
type TargetDef struct {
	ID       string         `json:"id"`
	Root     string         `json:"root"`
	MaxDepth map[string]int `json:"maxDepth,omitempty"`
	// KeepWhole lists lower-case extensions (no dot) that mark an archive as one unit: a folder-loader archive laying
	// out a file with one installs as a single entry holding every file it laid out, since such files only work
	// beside their siblings. An archive with none splits into one entry per file.
	KeepWhole []string `json:"keepWhole,omitempty"`
	// Role names a path role instead of a profile folder: the target's files are placed at install in that folder of
	// the game, shared by every profile, and Mortar records which it placed. Root is empty then.
	Role string `json:"role,omitempty"`
	// Extensions are the lower-case extensions (no dot) of files a folder-loader archive lays out in this target
	// rather than in mods.
	Extensions []string `json:"extensions,omitempty"`
}

// ProfileFolder is the folder below the profile's root that the target's files are laid out in: "" for {profile},
// the subfolder for {profile}/<sub>, and "mods" for {profileMods}.
func (t TargetDef) ProfileFolder() string {
	if t.Root == "{profileMods}" {
		return "mods"
	}
	return strings.TrimPrefix(strings.TrimPrefix(t.Root, "{profile}"), "/")
}

// MarkerPath is Marker as a path below the install root.
func (g GameInfo) MarkerPath() string { return path.Join(g.MarkerDir, g.Marker) }

// Target returns the game's content target with the given id.
func (g GameInfo) Target(id string) (TargetDef, bool) {
	for _, t := range g.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return TargetDef{}, false
}

// GameStores names a game to each store that sells it; a store that does not sell it is nil.
type GameStores struct {
	Steam   *SteamStore   `json:"steam,omitempty"`
	GOG     *GOGStore     `json:"gog,omitempty"`
	Lutris  *LutrisStore  `json:"lutris,omitempty"`
	Bottles *BottlesStore `json:"bottles,omitempty"`
	EA      *EAStore      `json:"ea,omitempty"`
}

// EAStore names a game to the EA App: the folder under an EA library folder (`EA Games`) its installer gives it.
type EAStore struct {
	Folder string `json:"folder"`
}

// BottlesStore names a game to Bottles: the folder its Steam and GOG installs have inside a bottle.
type BottlesStore struct {
	Folder string `json:"folder"`
}

// SteamStore names a game to Steam.
type SteamStore struct {
	AppID string `json:"appId"`
}

// GOGStore names a game to GOG: its product id and the folder GOG installers give it.
type GOGStore struct {
	ProductID string `json:"productId"`
	Folder    string `json:"folder"`
}

// LutrisStore names a game to Lutris: its slug and a word its executable paths contain.
type LutrisStore struct {
	Slug    string `json:"slug"`
	Keyword string `json:"keyword"`
}

// GameLoader is a mod loader a game can run.
type GameLoader struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// NexusModID is the loader's Nexus mod page, which Mortar installs outside any profile entry's Nexus source but
	// which is installed all the same.
	NexusModID int `json:"nexusModId,omitempty"`
	// Companion names the game's component (kind bridge) that Mortar installs alongside the loader so the running game
	// can talk to Mortar.
	Companion string `json:"companion,omitempty"`
	// Package is the Thunderstore package (Namespace-Name) the loader ships as, for a community that packs its own
	// build of it (Valheim's denikson-BepInExPack_Valheim); empty means DefaultLoaderPackage.
	Package string `json:"package,omitempty"`
}

// saveFilePattern reports whether p names files in the saves folder or in one plain folder below it; an exclusion
// ("!" first) names files in any of them by name alone.
func saveFilePattern(p string) bool {
	file, not := strings.CutPrefix(p, "!")
	parts := strings.Split(file, "/")
	if _, err := path.Match(file, ""); err != nil || strings.Contains(p, "\\") || len(parts) > 2 || not && len(parts) > 1 {
		return false
	}
	if len(parts) == 2 && strings.ContainsAny(parts[0], "*?[") {
		return false
	}
	return !slices.ContainsFunc(parts, func(s string) bool { return s == "" || s == "." || s == ".." })
}

// DefaultLoaderPackage is the BepInEx pack most Thunderstore communities share.
const DefaultLoaderPackage = "BepInEx-BepInExPack"

// LoaderPackage is the Thunderstore package the game's BepInEx loader ships as.
func (g GameInfo) LoaderPackage() string {
	for _, l := range g.Loaders {
		if l.Package != "" {
			return l.Package
		}
	}
	return DefaultLoaderPackage
}

// IsLoaderPackage reports whether the Thunderstore package id (Namespace-Name) is a BepInEx pack some catalog game's
// loader installs, which the loader provides and no profile entry stands for. Pack names are unique across
// communities, so the whole catalog answers without knowing the game.
func IsLoaderPackage(id string) bool {
	if strings.EqualFold(id, DefaultLoaderPackage) {
		return true
	}
	for _, g := range Games() {
		if strings.EqualFold(id, g.LoaderPackage()) {
			return true
		}
	}
	return false
}

// GameSource is a site the game's mods come from. Key is the site's name for the game (Nexus domain, Thunderstore
// community) and GameID its numeric id where the site's API uses one.
type GameSource struct {
	ID     string `json:"id"`
	Key    string `json:"key,omitempty"`
	GameID int    `json:"gameId,omitempty"`
	// Classes are the numeric class ids a search covers, for a game whose mods sit in several (CurseForge's Create a
	// Sim and Build / Buy beside Mods); it lists Key, the class the source is addressed by.
	Classes []string `json:"classes,omitempty"`
	// Loaders and GameVersions narrow an update check to the files for the game's mod loaders and game versions, on a
	// site whose files carry them (Modrinth).
	Loaders      []string `json:"loaders,omitempty"`
	GameVersions []string `json:"gameVersions,omitempty"`
}

// Source returns the game's source with the given id.
func (g GameInfo) Source(id string) (GameSource, bool) {
	for _, s := range g.Sources {
		if s.ID == id {
			return s, true
		}
	}
	return GameSource{}, false
}

// NexusDomain is the game's Nexus v1 URL segment, empty when Nexus does not host it.
func (g GameInfo) NexusDomain() string {
	s, _ := g.Source("nexus")
	return s.Key
}

// NexusID is the game's numeric Nexus id in the v2 API, 0 when Nexus does not host it.
func (g GameInfo) NexusID() int {
	s, _ := g.Source("nexus")
	return s.GameID
}

// LoaderNexusModID is the Nexus mod page of the game's first loader, 0 when it has none.
func (g GameInfo) LoaderNexusModID() int {
	if len(g.Loaders) == 0 {
		return 0
	}
	return g.Loaders[0].NexusModID
}

// SteamAppID is the game's Steam app id, empty when Steam does not sell it.
func (g GameInfo) SteamAppID() string {
	if g.Stores.Steam == nil {
		return ""
	}
	return g.Stores.Steam.AppID
}

// Validate checks that a game names itself, has a loader, and that no file or folder name could leave its folder.
func (g GameInfo) Validate() error {
	if g.ID == "" || g.Name == "" || g.Marker == "" {
		return errors.New("a game needs an id, a name and a marker file")
	}
	if len(g.Loaders) == 0 {
		return fmt.Errorf("game %q needs a loader", g.ID)
	}
	if g.Enabled && len(g.Targets) == 0 {
		return fmt.Errorf("enabled game %q needs a content target", g.ID)
	}
	if g.Enabled && g.Stores == (GameStores{}) {
		return fmt.Errorf("enabled game %q needs a store", g.ID)
	}
	names := []string{g.Marker}
	if g.LinuxMarker != "" {
		names = append(names, g.LinuxMarker)
	}
	if g.Stores.GOG != nil {
		names = append(names, g.Stores.GOG.Folder)
	}
	if g.Stores.EA != nil {
		names = append(names, g.Stores.EA.Folder)
	}
	for _, name := range names {
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			return fmt.Errorf("game %q has an unsafe file or folder name %q", g.ID, name)
		}
	}
	if g.MarkerDir != "" && (strings.Contains(g.MarkerDir, "\\") || !fs.ValidPath(g.MarkerDir) || g.MarkerDir == ".") {
		return fmt.Errorf("game %q has an unsafe marker folder %q", g.ID, g.MarkerDir)
	}
	for _, r := range g.RequiredSettings {
		if _, ok := g.Paths[r.Path]; !ok || r.Key == "" || r.Value == "" || r.Message == "" {
			return fmt.Errorf("game %q has a required setting without a path role of the game, a key, a value or a message", g.ID)
		}
	}
	for _, k := range g.KnownBroken {
		if err := k.Validate(); err != nil {
			return fmt.Errorf("game %q: %w", g.ID, err)
		}
	}
	if g.Graphics != nil {
		if err := g.Graphics.Validate(); err != nil {
			return fmt.Errorf("game %q: %w", g.ID, err)
		}
	}
	for _, p := range g.SaveFiles {
		if !saveFilePattern(p) {
			return fmt.Errorf("game %q has an unusable save file pattern %q", g.ID, p)
		}
	}
	for _, ext := range g.SaveCompanions {
		if len(ext) < 2 || ext[0] != '.' || strings.ContainsAny(ext, "/\\*?[") {
			return fmt.Errorf("game %q has an unusable save companion %q", g.ID, ext)
		}
	}
	for _, cp := range g.Caches {
		if _, ok := g.Paths[cp.Role]; !ok {
			return fmt.Errorf("game %q cache %q names path role %q with no such path", g.ID, cp.Path, cp.Role)
		}
		if cp.Path == "" || strings.Contains(cp.Path, "\\") || !fs.ValidPath(cp.Path) || cp.Path == "." {
			return fmt.Errorf("game %q cache %q on role %q is not a local slash path", g.ID, cp.Path, cp.Role)
		}
	}
	for role, t := range g.Paths {
		for _, p := range []string{t.Windows, t.Linux, t.Darwin} {
			if p != "" && !strings.HasPrefix(p, "{") {
				return fmt.Errorf("game %q path %q must start with a token", g.ID, role)
			}
		}
	}
	if g.Deploy != DeployRedirect && g.Deploy != DeployProfile {
		return fmt.Errorf("game %q has unknown deploy method %q", g.ID, g.Deploy)
	}
	targets := make(map[string]struct{}, len(g.Targets))
	for _, t := range g.Targets {
		if t.Role != "" {
			if _, ok := g.Paths[t.Role]; !ok || t.Root != "" {
				return fmt.Errorf("game %q target %q names path role %q with no such path, or also a root", g.ID, t.ID, t.Role)
			}
		} else if t.ID == "" || !underToken(t.Root, "{profileMods}", "{profile}") {
			return fmt.Errorf("game %q has a target without an id or with an unknown root %q", g.ID, t.Root)
		}
		for _, ext := range t.Extensions {
			if ext == "" || ext != strings.ToLower(ext) || strings.ContainsAny(ext, ".\\/") {
				return fmt.Errorf("game %q target %q has an unusable extension %q", g.ID, t.ID, ext)
			}
		}
		if _, ok := targets[t.ID]; ok {
			return fmt.Errorf("game %q lists target %q more than once", g.ID, t.ID)
		}
		for _, ext := range t.KeepWhole {
			if ext == "" || ext != strings.ToLower(ext) || strings.ContainsAny(ext, ".\\/") {
				return fmt.Errorf("game %q target %q has an unusable keepWhole extension %q", g.ID, t.ID, ext)
			}
		}
		targets[t.ID] = struct{}{}
	}
	sources := make(map[string]struct{}, len(g.Sources))
	for _, s := range g.Sources {
		if s.ID == "" {
			return fmt.Errorf("game %q has a source without an id", g.ID)
		}
		if _, ok := sources[s.ID]; ok {
			return fmt.Errorf("game %q lists source %q more than once", g.ID, s.ID)
		}
		sources[s.ID] = struct{}{}
		if len(s.Classes) > 0 && !slices.Contains(s.Classes, s.Key) {
			return fmt.Errorf("game %q source %q lists classes without its key %q", g.ID, s.ID, s.Key)
		}
		for _, c := range s.Classes {
			if _, err := strconv.ParseUint(c, 10, 31); err != nil {
				return fmt.Errorf("game %q source %q has a non-numeric class %q", g.ID, s.ID, c)
			}
		}
	}
	return g.validateTemplates(sources)
}

func (g GameInfo) validateTemplates(sources map[string]struct{}) error {
	seen := map[string]struct{}{}
	for _, t := range g.Templates {
		if t.ID == "" || t.Name == "" || len(t.Packages) == 0 {
			return fmt.Errorf("game %q has a template without an id, a name or packages", g.ID)
		}
		if _, ok := seen[t.ID]; ok {
			return fmt.Errorf("game %q lists template %q more than once", g.ID, t.ID)
		}
		seen[t.ID] = struct{}{}
		for _, p := range t.Packages {
			if _, ok := sources[p.Source]; !ok || p.Ref == "" || p.Name == "" {
				return fmt.Errorf("game %q template %q has a package with no known source, ref or name", g.ID, t.ID)
			}
			if p.Source == "nexus" {
				if n, err := strconv.Atoi(p.Ref); err != nil || n < 1 {
					return fmt.Errorf("game %q template %q has Nexus ref %q that is not a mod id", g.ID, t.ID, p.Ref)
				}
			}
		}
	}
	return nil
}

// underToken reports a path that is one of the tokens or a folder below one.
func underToken(p string, tokens ...string) bool {
	for _, t := range tokens {
		if p == t {
			return true
		}
		if rest, ok := strings.CutPrefix(p, t+"/"); ok && rest != "" && !strings.Contains(rest, "..") && !strings.Contains(rest, "\\") {
			return true
		}
	}
	return false
}

// Component is one resolved, hashed asset in a manifest.
type Component struct {
	Game     string `json:"game"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Source   Source `json:"source"`
	Tag      string `json:"tag"`
	Asset    string `json:"asset"`
	Version  string `json:"version"`
	Accepted string `json:"accepted,omitempty"`
	SHA256   string `json:"sha256"`
}

// Manifest is the signed set of component assets Mortar may install.
type Manifest struct {
	Serial     uint64      `json:"serial"`
	Components []Component `json:"components"`
	Games      []GameInfo  `json:"games"`
}

// Validate checks the source host, component identity and asset digest.
func (s Source) Validate() error {
	switch strings.ToLower(strings.TrimSpace(s.Host)) {
	case "github.com":
		if !validPart(s.Owner) || !validPart(s.Repo) {
			return errors.New("GitHub component source needs an owner and repository")
		}
	case "thunderstore.io":
		if !validPart(s.Namespace) || !validPart(s.Name) {
			return errors.New("thunderstore component source needs a namespace and name")
		}
	default:
		return fmt.Errorf("component source host %q is not allowed", s.Host)
	}
	return nil
}

func validPart(value string) bool {
	return value != "" && !strings.ContainsAny(value, "/\\?#")
}

// Validate checks a resolved component.
func (c Component) Validate() error {
	if c.Game == "" || c.Name == "" {
		return errors.New("component game and name are required")
	}
	if c.Kind != "loader" && c.Kind != "bridge" {
		return fmt.Errorf("component %q has unknown kind %q", c.Name, c.Kind)
	}
	if c.Version == "" || c.Asset == "" || c.Tag == "" {
		return fmt.Errorf("component %q is missing its resolved release", c.Name)
	}
	if err := c.Source.Validate(); err != nil {
		return fmt.Errorf("component %q: %w", c.Name, err)
	}
	if len(c.SHA256) != sha256.Size*2 {
		return fmt.Errorf("component %q has an invalid sha256", c.Name)
	}
	if _, err := hex.DecodeString(c.SHA256); err != nil {
		return fmt.Errorf("component %q has an invalid sha256: %w", c.Name, err)
	}
	return nil
}

// Validate checks every component and rejects duplicate game/name pairs.
func (m Manifest) Validate() error {
	if m.Serial == 0 {
		return errors.New("component manifest has no serial")
	}
	seen := make(map[string]struct{}, len(m.Components))
	for _, c := range m.Components {
		if err := c.Validate(); err != nil {
			return err
		}
		key := c.Game + "\x00" + c.Name
		if _, ok := seen[key]; ok {
			return fmt.Errorf("component %q is listed more than once for %s", c.Name, c.Game)
		}
		seen[key] = struct{}{}
	}
	games := make(map[string]struct{}, len(m.Games))
	for _, g := range m.Games {
		if err := g.Validate(); err != nil {
			return err
		}
		if _, ok := games[g.ID]; ok {
			return fmt.Errorf("game %q is listed more than once", g.ID)
		}
		games[g.ID] = struct{}{}
	}
	return nil
}

// Decode parses and validates a manifest after its signature has been checked.
func Decode(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("decode component manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	enableForSandbox(m.Games)
	return m, nil
}

// enableGamesEnv lists catalog games, comma separated, to switch on whatever the manifest says. A self-test sandbox
// sets it to try a game that has not shipped; the shipped catalog is never edited for that.
const enableGamesEnv = "MORTAR_ENABLE_GAMES"

func enableForSandbox(games []GameInfo) {
	ids := strings.Split(os.Getenv(enableGamesEnv), ",")
	for i := range games {
		if slices.Contains(ids, games[i].ID) {
			games[i].Enabled = true
		}
	}
}

// Verify checks a detached Ed25519 signature without parsing the manifest.
func Verify(manifest, signature, publicKey []byte) error {
	key, err := parsePublicKey(publicKey)
	if err != nil {
		return err
	}
	sig, err := parseSignature(signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, manifest, sig) {
		return errors.New("component manifest signature is invalid")
	}
	return nil
}

func parsePublicKey(data []byte) (ed25519.PublicKey, error) {
	if len(data) == ed25519.PublicKeySize {
		return ed25519.PublicKey(data), nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("component public key is not PEM")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse component public key: %w", err)
	}
	public, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("component public key is not Ed25519")
	}
	return public, nil
}

func parseSignature(data []byte) ([]byte, error) {
	if len(data) == ed25519.SignatureSize {
		return data, nil
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == ed25519.SignatureSize {
		return trimmed, nil
	}
	for _, encoding := range []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding} {
		sig, err := encoding.DecodeString(string(trimmed))
		if err == nil && len(sig) == ed25519.SignatureSize {
			return sig, nil
		}
	}
	return nil, errors.New("component signature is not an Ed25519 signature")
}

// bundled is the decoded manifest compiled into Mortar, remembered with the sandbox switch it was decoded under. It is
// shared like the active client's manifest: callers read it and never write into it.
var bundled struct {
	mu     sync.Mutex
	env    string
	loaded bool
	m      Manifest
	err    error
}

// BundledManifest returns the manifest compiled into Mortar.
func BundledManifest() (Manifest, error) {
	env := os.Getenv(enableGamesEnv)
	bundled.mu.Lock()
	defer bundled.mu.Unlock()
	if !bundled.loaded || bundled.env != env {
		bundled.m, bundled.err = Decode(bundledJSON)
		bundled.env, bundled.loaded = env, true
	}
	return bundled.m, bundled.err
}

// BundledAccepted is the game versions the bundled loader component of game accepts (">=1.6.14"), empty when it
// declares none.
func BundledAccepted(game, loader string) string {
	m, err := BundledManifest()
	if err != nil {
		return ""
	}
	c, _ := findComponent(m.Components, game, loader)
	return c.Accepted
}

type cachedManifest struct {
	Manifest  []byte `json:"manifest"`
	Signature []byte `json:"signature"`
}

// Client loads the signed manifest and downloads assets named by it.
type Client struct {
	HTTP *http.Client
	// ManifestURL, when set, is fetched as is instead of looking up the newest components release.
	ManifestURL   string
	ReleasesURL   string
	GitHubBaseURL string
	manifest      Manifest
}

// NewClient returns a component client using hc for manifest and asset requests.
func NewClient(hc *http.Client) *Client {
	return &Client{HTTP: hc}
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) manifestURL(ctx context.Context) (string, error) {
	if c.ManifestURL != "" {
		return c.ManifestURL, nil
	}
	body, err := c.get(ctx, cmp.Or(c.ReleasesURL, ReleasesURL), maxManifest)
	if err != nil {
		return "", err
	}
	var releases []struct {
		Tag    string `json:"tag_name"`
		Draft  bool   `json:"draft"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", err
	}
	for _, r := range releases {
		if r.Draft || !strings.HasPrefix(r.Tag, tagPrefix) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == "components.json" {
				return a.URL, nil
			}
		}
	}
	return "", errors.New("no components release is published")
}

// SetManifest selects the already verified manifest used by Component and Download.
func (c *Client) SetManifest(m Manifest) {
	c.manifest = m
}

// Manifest returns the currently selected manifest.
func (c *Client) Manifest() Manifest {
	return c.manifest
}

// Component returns a component by game and name.
func (c *Client) Component(game, name string) (Component, bool) {
	if component, ok := findComponent(c.manifest.Components, game, name); ok {
		return component, true
	}
	// Before Load (and when the fetched manifest lacks it) the manifest compiled into Mortar answers, as Game does,
	// so startup work that runs ahead of the network fetch still finds its components.
	m, err := BundledManifest()
	if err != nil {
		return Component{}, false
	}
	return findComponent(m.Components, game, name)
}

func findComponent(components []Component, game, name string) (Component, bool) {
	for _, component := range components {
		if component.Game == game && component.Name == name {
			return component, true
		}
	}
	return Component{}, false
}

// Game returns a game's identity from the selected manifest, else from the bundled one, so a manifest that lacks a
// game never leaves Mortar without it.
func (c *Client) Game(id string) (GameInfo, bool) {
	if g, ok := findGame(c.manifest.Games, id); ok {
		return g, true
	}
	return bundledGame(id)
}

func bundledGame(id string) (GameInfo, bool) {
	m, err := BundledManifest()
	if err != nil {
		return GameInfo{}, false
	}
	return findGame(m.Games, id)
}

func findGame(games []GameInfo, id string) (GameInfo, bool) {
	for _, g := range games {
		if g.ID == id {
			return g, true
		}
	}
	return GameInfo{}, false
}

// ErrBadSignature and ErrRollback are the Load failures that point at tampering rather than an unreachable or
// unreadable manifest: one whose signature does not verify, and one older than the manifest Mortar already trusts.
var (
	ErrBadSignature = errors.New("component manifest signature does not verify")
	ErrRollback     = errors.New("component manifest is older than the one already trusted")
)

// Load fetches the signed manifest at startup, using the existing meta cache for a daily TTL. A failed
// fetch or invalid cached entry falls back to the bundled manifest.
func (c *Client) Load(ctx context.Context, cache *meta.Client, publicKey []byte) (Manifest, error) {
	bundled, bundledErr := BundledManifest()
	if bundledErr != nil {
		return Manifest{}, bundledErr
	}
	// A fetched manifest older than the one this build ships or the one already cached is a rollback.
	oldSerial := bundled.Serial
	if cache != nil {
		if old, ok := meta.Peek[cachedManifest](cache, cacheName); ok {
			if err := Verify(old.Manifest, old.Signature, publicKey); err == nil {
				if parsed, err := Decode(old.Manifest); err == nil {
					oldSerial = max(oldSerial, parsed.Serial)
				}
			}
		}
	}
	var fetchFailure error
	fetch := func() (cachedManifest, error) {
		url, manifest, signature, err := c.fetch(ctx)
		if err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		if err := Verify(manifest, signature, publicKey); err != nil {
			fetchFailure = fmt.Errorf("%w: %s: %w", ErrBadSignature, url, err)
			return cachedManifest{}, fetchFailure
		}
		parsed, err := Decode(manifest)
		if err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		if parsed.Serial < oldSerial {
			fetchFailure = fmt.Errorf("%w: %s has serial %d, below %d", ErrRollback, url, parsed.Serial, oldSerial)
			return cachedManifest{}, fetchFailure
		}
		return cachedManifest{Manifest: manifest, Signature: signature}, nil
	}

	var entry cachedManifest
	var err error
	if cache != nil {
		entry, err = meta.CachedForBuild(cache, cacheName, day, fetch)
	} else {
		entry, err = fetch()
	}
	if err == nil {
		if verifyErr := Verify(entry.Manifest, entry.Signature, publicKey); verifyErr == nil {
			if parsed, parseErr := Decode(entry.Manifest); parseErr == nil {
				c.SetManifest(parsed)
				return parsed, fetchFailure
			} else {
				err = parseErr
			}
		} else {
			err = fmt.Errorf("%w: cached copy: %w", ErrBadSignature, verifyErr)
		}
	}
	c.SetManifest(bundled)
	return bundled, err
}

// fetch returns the manifest's URL, which names its release, with the manifest and its signature.
func (c *Client) fetch(ctx context.Context) (string, []byte, []byte, error) {
	url, err := c.manifestURL(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	manifest, err := c.get(ctx, url, maxManifest)
	if err != nil {
		return url, nil, nil, err
	}
	signature, err := c.get(ctx, url+".sig", maxSignature)
	if err != nil {
		return url, nil, nil, err
	}
	return url, manifest, signature, nil
}

func (c *Client) get(ctx context.Context, address string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("component manifest request returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("component manifest response is too large")
	}
	return body, nil
}

// Download fetches a component into dest and verifies its SHA-256 before replacing dest. A failed or
// mismatched download never changes an existing dest.
func (c *Client) Download(ctx context.Context, component Component, dest string) error {
	if err := component.Validate(); err != nil {
		return err
	}
	if strings.ToLower(component.Source.Host) != "github.com" {
		return fmt.Errorf("component source host %q is not supported for downloads", component.Source.Host)
	}
	tag := cmp.Or(component.Tag, component.Version)
	base := cmp.Or(c.GitHubBaseURL, "https://github.com")
	address := strings.TrimRight(base, "/") + "/" + component.Source.Owner + "/" + component.Source.Repo +
		"/releases/download/" + tag + "/" + component.Asset
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".mortar-component-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer func() {
		_ = os.Remove(tmpPath)
		_ = os.Remove(github.ResumeSidecar(tmpPath))
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := github.Download(ctx, c.HTTP, address, tmpPath, maxComponent, nil); err != nil {
		return fmt.Errorf("download component %s: %w", component.Name, err)
	}
	sum, err := fsx.SHA256(tmpPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(sum, component.SHA256) {
		return fmt.Errorf("component %s has sha256 %s, want %s", component.Name, sum, component.SHA256)
	}
	return datadir.WriteStream(dest, 0o600, func(w io.Writer) error {
		f, err := fsx.Open(tmpPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		_, err = io.Copy(w, f)
		return err
	})
}

// HasMetadata reports whether the game uses the named metadata feature.
func (g GameInfo) HasMetadata(name string) bool { return slices.Contains(g.Metadata, name) }
