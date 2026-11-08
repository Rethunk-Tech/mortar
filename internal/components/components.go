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
	"log/slog"
	"math"
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

	"github.com/Rethunk-Tech/go-bepinex"

	gh "github.com/Rethunk-Tech/go-github"
	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-httpkit/download"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-semverish"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
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
	// floorName holds the highest serial accepted, apart from the cache entry so an expired entry cannot reopen the way
	// to an older signed catalog.
	floorName = "serial-floor-" + cacheName
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

// AssetName is the component's asset pattern with its placeholders filled for a release tag: {version} is the tag
// without one leading v, {tag} is the tag as the release spells it.
func (c SourceComponent) AssetName(tag string) string {
	name := strings.ReplaceAll(c.AssetPattern, "{version}", strings.TrimPrefix(tag, "v"))
	return strings.ReplaceAll(name, "{tag}", tag)
}

// SourceFile is the committed input read by cmd/components.
type SourceFile struct {
	Components []SourceComponent `json:"components"`
	Games      []GameInfo        `json:"games"`
	Authors    []AuthorAliases   `json:"authors,omitempty"`
}

// versionConstraint is one operator and version of a known-broken entry's GameVersions.
var versionConstraint = regexp.MustCompile(`^(<=|>=|<|>|=)\S+$`)

// GameInfo is one game's catalog entry: the names and ids the stores and mod sites know it by, its loaders and its mod
// sources. How a game is launched or modded stays in its Go implementation; this is only what can change without a
// Mortar release.
type GameInfo struct {
	// ID is the slug. It is permanent: profiles, settings and share links store it.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Enabled is false for a game listed as coming later.
	Enabled bool `json:"enabled"`
	// MinVersion is the oldest Mortar that can run the entry, for one that needs driver code an older build lacks; an
	// older build leaves the game out rather than listing a game it cannot set up.
	MinVersion string `json:"minVersion,omitempty"`
	// ImportIDs are the names Vortex and Mod Organizer 2 give the game, so their profiles can be imported; a game
	// neither manager has an extension or plugin for has none.
	ImportIDs ImportIDs `json:"importIds,omitzero"`
	// Marker is a file every install of the game holds, at its root or one "game" folder down. A .exe marker makes a
	// Windows build.
	Marker string `json:"marker"`
	// MarkerDir is the slash-separated folder below the install root that holds Marker, for a game whose executable
	// sits below the root (Game/Bin); empty when the marker is at the root or one "game" folder down.
	MarkerDir string `json:"markerDir,omitempty"`
	// LinuxMarker is the executable of the game's native Linux build where the store ships one beside the build Marker
	// names: an install holding it and not Marker is that build, started natively rather than under Proton.
	LinuxMarker string `json:"linuxMarker,omitempty"`
	// DarwinMarker is the slash path below the install folder of the executable inside the game's macOS bundle
	// (Game.app/Contents/MacOS/Game). On a Mac an install holding it and not Marker is that build. A store that
	// installs the .app itself needs no entry: the bundle's folder is the install.
	DarwinMarker string `json:"darwinMarker,omitempty"`
	// Processes are other executables in the install the game runs as (another renderer's build), file names only, so
	// running-game detection sees them too.
	Processes []string `json:"processes,omitempty"`
	// LinuxStart is the slash path below the install of the file the native Linux build is started through, where
	// that is not LinuxMarker (a script that sets the build's environment first).
	LinuxStart string `json:"linuxStart,omitempty"`
	// Art is the slash path below the install of an image the game ships, shown as its art where no store has any.
	Art string `json:"art,omitempty"`
	// R2modmanFolder is r2modman's own folder name for the game (the Thunderstore schema's internalFolderName), the
	// parent of its profiles folder.
	R2modmanFolder string `json:"r2modmanFolder,omitempty"`
	// Metadata names the SMAPI-derived features that apply to the game: smapi-updates, smapi-compat, stardew-dataset.
	Metadata []string `json:"metadata"`
	// NewProfileSeparateSaves starts a new profile of this game with its own saves rather than the shared ones.
	NewProfileSeparateSaves bool `json:"newProfileSeparateSaves,omitempty"`
	// Paths names folders and files outside the install by role, one PathTemplate per platform. Roles: saves,
	// errorLogs, startupPreferences; mods (a folder game's mod folder, filled for a launch, then emptied with the
	// player's files returned); options (one settings file: each profile holds its own copy, seeded on first launch and
	// swapped in per launch); tray (shared by every profile, filled by a target with a Role); userData (the base
	// Caches paths resolve under). Any other role a RequiredSettings or Caches entry names must be listed too.
	Paths map[string]PathTemplate `json:"paths,omitempty"`
	// SaveFiles are patterns naming each save file in the saves folder or one folder below it, for a game that keeps a
	// save as a file; a pattern starting with "!" excludes the files it matches. Without them a save is a folder holding
	// a file of its own name, as Stardew Valley's are.
	SaveFiles []string `json:"saveFiles,omitempty"`
	// SaveCompanions are extensions of files beside a save file, sharing its stem, that belong to that save (a Valheim
	// world's .db beside its .fwl).
	SaveCompanions []string `json:"saveCompanions,omitempty"`
	// SaveCompanionPaths are slash paths under the saves folder, files or folders, that belong to one save: {stem} is
	// the save file's name without its extension (a Cat Mail Co. save's CatLib mod data, "CatLibSaves/{stem}"). A
	// backup carries them and a restore puts back those it holds. An older Mortar ignores the field.
	SaveCompanionPaths []string `json:"saveCompanionPaths,omitempty"`
	// SaveCopies are patterns for the copies of a save the game keeps itself beside it, which restore over the save:
	// {name} is the save's file name, {stem} the name without its extension and {ext} the extension (a Valheim world's
	// "{stem}_backup_auto-*{ext}", a Sims 4 slot's "{name}.ver?"). An older Mortar ignores the field.
	SaveCopies []string `json:"saveCopies,omitempty"`
	// SaveCopyFolders are folders beside the saves folder where the game also keeps copies named by SaveCopies
	// (Vintage Story's Backups and BackupSaves). An older Mortar ignores the field.
	SaveCopyFolders []string `json:"saveCopyFolders,omitempty"`
	// SaveLinked are slash paths beside the saves folder of files that belong to one save, {uuid} standing for the
	// world's own identifier (Vintage Story's minimap, "Maps/{uuid}.db"); backups and restores carry them with the save.
	// An older Mortar ignores the field.
	SaveLinked []string `json:"saveLinked,omitempty"`
	// SaveUserFolders means the saves folder holds a folder per platform user with the SaveFiles inside each (Cities:
	// Skylines II's Saves/<Steam id>/<city>.cok). An older Mortar ignores the field and finds no saves.
	SaveUserFolders bool `json:"saveUserFolders,omitempty"`
	// FolderNames lists, by a folder name Paths uses, the names the game gives that folder on some installs (The Sims
	// 4 names its user folder after the game's language). An older Mortar ignores the field and uses the name in Paths.
	FolderNames map[string][]string `json:"folderNames,omitempty"`
	// RequiredSettings are settings in a game's own files that mods need switched on; Problems reports one that is
	// switched off and never edits the file. An older Mortar ignores the field.
	RequiredSettings []RequiredSetting `json:"requiredSettings,omitempty"`
	// Caches are rebuildable cache files or folders the game rebuilds on launch, which Mortar deletes once after a
	// profile's mod set changes.
	Caches []CachePath `json:"caches,omitempty"`
	// TitleScene is the Unity scene of the game's main menu, where a BepInEx startup measurement ends; without it the
	// first scene ends it.
	TitleScene string `json:"titleScene,omitempty"`
	// ReadySignal is the text of the Unity log message the game prints when its own loading is done, which can be well
	// after the title scene shows; the BepInEx startup measurement then ends there. An older Mortar ignores the field,
	// and an older bridge ends at the title scene.
	ReadySignal string `json:"readySignal,omitempty"`
	// Graphics offers the player a choice of graphics API for the game, each passed as launch arguments; an older
	// Mortar ignores the field.
	Graphics *Graphics `json:"graphics,omitempty"`
	// ModsArg is the launch argument the folder loader passes the profile's mods folder under (r2modman's Ancient
	// Dungeon VR reads "--mods-path <folder>"), for a game that takes its mods from a folder named on the command
	// line: nothing is placed, so the game needs no mods path role. An older Mortar ignores the field, so an entry
	// using it sets MinVersion.
	ModsArg string `json:"modsArg,omitempty"`
	// Deploy is how the profile reaches the game: redirect (the loader points the game at the profile's mods folder,
	// nothing is placed) or profile (the profile holds the loader and its mods, and the loader's install-side files are
	// placed into the install for the launch and taken back after).
	Deploy string `json:"deploy"`
	// Targets are the places the game's mod files go.
	Targets []TargetDef `json:"targets"`
	// Stores holds the game's id per store driver key; a game no store app tracks sets ManualFolder instead.
	Stores GameStores `json:"stores"`
	// ManualFolder marks a game sold where no store app tracks it (an itch.io download unpacked anywhere): it names no
	// store, and the player browses to its folder, as for a game they add themselves.
	ManualFolder bool `json:"manualFolder,omitempty"`
	// Loaders are the loader ids the game can run (the catalog test's knownLoaders names them; "folder" is a game with
	// none). A Thunderstore-installed loader names its Package. A companion bridge installs only from the game's own
	// kind: bridge component, never Thunderstore, so a game without one gets no startup timing, in-game performance,
	// stream overlay, intro skip or console commands even when its loader names a Companion. MelonLoader without
	// Routes takes the 0.7 recursive layout; with them, the route marked Default takes a file nothing else claims.
	// Keep r2modman's rule order when copying its install rules into Routes.
	Loaders []GameLoader `json:"loaders"`
	// Sources are the mod sites, in browse order. A link and handoff source (patreon, itch) is just {"id": ...}: browse
	// does not search it, a pasted address resolves to it and Mortar opens the page.
	Sources []GameSource `json:"sources"`
	// Templates are starter profiles offered when creating one: a named list of packages, each queued through the
	// normal download path.
	Templates []StarterTemplate `json:"templates,omitempty"`
	// Server marks a dedicated server: the game it runs mods of and the launch arguments its profile settings fill.
	// An older Mortar leaves the entry out through MinVersion.
	Server *ServerSpec `json:"server,omitempty"`
	// ProfileServer marks a game whose profile can run the game's dedicated server instead of the client, from the
	// same install and with the profile's mods. An older Mortar leaves the entry out through MinVersion.
	ProfileServer *ProfileServerSpec `json:"profileServer,omitempty"`
}

// ProfileServerSpec describes how a profile starts the game's dedicated server from the game's install folder.
type ProfileServerSpec struct {
	// Exe is the server's executable in the install, started as it is on Windows.
	Exe string `json:"exe"`
	// Assembly is the .NET assembly of the server in the install, which every other platform runs with dotnet.
	Assembly string `json:"assembly"`
	// DataFlag is the argument that precedes the profile's server data folder, where the server keeps its config,
	// saves, logs and the Mods and ModConfig the profile gives it.
	DataFlag string `json:"dataFlag"`
}

// ServerSpec describes a dedicated server's launch.
type ServerSpec struct {
	// Clients is the id of the game whose profiles' mod lists the server runs, so a server profile can be filled from
	// one of them.
	Clients string `json:"clients"`
	// Fixed are arguments every launch passes (-batchmode, -nographics).
	Fixed []string `json:"fixed,omitempty"`
	// ConfigFile is the slash path below the install of a file Mortar writes from the settings that have a Cvar, for a
	// server configured by file rather than command line.
	ConfigFile string `json:"configFile,omitempty"`
	// Args are the arguments a profile's server settings fill, in order.
	Args []ServerArg `json:"args,omitempty"`
}

// ServerArg is one launch argument taken from a profile's server settings.
type ServerArg struct {
	// ID names the setting in the profile (server.json) and on the command line of `profile set`.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Flag precedes the value as its own argument ("-world"); Cvar instead names a line of the server's ConfigFile.
	// Exactly one is set.
	Flag string `json:"flag,omitempty"`
	Cvar string `json:"cvar,omitempty"`
	// Default is the value used while the profile sets none; an argument with neither a default nor Required is left out.
	Default string `json:"default,omitempty"`
	// Required makes a launch without a value fail, naming the setting.
	Required bool `json:"required,omitempty"`
	// Hint is a line of help under the field in the profile's server settings.
	Hint string `json:"hint,omitempty"`
	// Secret keeps the value out of messages and out of the settings form.
	Secret bool `json:"secret,omitempty"`
	// Pattern is a regular expression the whole value must match.
	Pattern string `json:"pattern,omitempty"`
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
	// ModRoot is the file name that marks a mod's own folder in a folder-loader archive (Vintage Story's
	// modinfo.json): the folder holding it is laid out whole as one folder below the target, and the archive is one
	// entry. An archive without it is laid out as loose files.
	ModRoot string `json:"modRoot,omitempty"`
	// Role names a path role instead of a profile folder: the target's files are placed at install in that folder of
	// the game, shared by every profile, and Mortar records which it placed. Root is empty then.
	Role string `json:"role,omitempty"`
	// Extensions are the lower-case extensions (no dot) of files a folder-loader archive lays out in this target
	// rather than in mods.
	Extensions []string `json:"extensions,omitempty"`
	// Sources are the source ids whose archives are laid out whole in this target, whatever the game's loader makes of
	// them (Paradox Mods archives go to the game's own Mods folder, not into the loader). Only a Role target takes
	// them.
	Sources []string `json:"sources,omitempty"`
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
	Epic    *EpicStore    `json:"epic,omitempty"`
	Xbox    *XboxStore    `json:"xbox,omitempty"`
	// Standalone and Flatpak find a game sold outside any store app: in the folders its own installer uses, or as a
	// Flatpak app.
	Standalone *StandaloneStore `json:"standalone,omitempty"`
	Flatpak    *FlatpakStore    `json:"flatpak,omitempty"`
}

// StandaloneStore names the folders a game's own installer puts it in, each a path template per platform.
type StandaloneStore struct {
	Folders []PathTemplate `json:"folders"`
}

// FlatpakStore names a game installed as a Flatpak app: the app id and the slash path below the app's files folder
// that holds the game.
type FlatpakStore struct {
	App    string `json:"app"`
	Folder string `json:"folder"`
}

var flatpakApp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*(\.[A-Za-z0-9_-]+){2,}$`)

// PlainFlatpakApp reports whether app is a Flatpak application id: three or more dot-separated parts of letters,
// digits, "_" and "-".
func PlainFlatpakApp(app string) bool { return len(app) <= 255 && flatpakApp.MatchString(app) }

// XboxStore names a game to the Xbox app: the Name of the Windows app package it installs (Get-AppxPackage -Name).
type XboxStore struct {
	Package string `json:"package"`
}

var plainPackage = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]{0,99}$`)

// PlainPackage reports whether name is an app package Name safe to put in a PowerShell query: letters, digits, "."
// and "-".
func PlainPackage(name string) bool { return plainPackage.MatchString(name) }

var plainAppName = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// PlainAppName reports whether name is an Epic AppName safe to put in a launcher link: letters, digits, "-" and "_".
func PlainAppName(name string) bool { return plainAppName.MatchString(name) }

// EpicStore names a game to the Epic Games Launcher: the AppName (or names, for a game sold as several products) its
// install list gives the game's installs.
type EpicStore struct {
	AppNames []string `json:"appNames"`
}

// EAStore names a game to the EA App: the folder under an EA library folder (`EA Games`) its installer gives it, and
// the HKLM key (SOFTWARE\...) whose "Install Dir" its installer writes, which finds a game installed on another drive.
type EAStore struct {
	Folder   string `json:"folder"`
	Registry string `json:"registry,omitempty"`
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
	// Console is whether the game has a command console its companion can run lines in (Valheim's), which the Console
	// tab's input needs on a BepInEx game.
	Console bool `json:"console,omitempty"`
	// Package is the Thunderstore package (Namespace-Name) the loader ships as, for a community that packs its own
	// build of it (Valheim's denikson-BepInExPack_Valheim); empty means DefaultLoaderPackage.
	Package string `json:"package,omitempty"`
	// Major is the loader's major version the game runs, when it is not the loader's own default: BepInEx 6 for an
	// IL2CPP game such as Cat Mail Co., whose pack is BepInEx-BepInExPack_IL2CPP.
	Major int `json:"major,omitempty"`
	// Routes are folders beyond BepInEx's own that the game's mods install into, from r2modman's install rules.
	Routes []LoaderRoute `json:"routes,omitempty"`
	// Proxy is the Doorstop proxy library the game's BepInEx pack ships when it is not winhttp.dll (In Silence's pack
	// renames it to version.dll); one of bepinex.Proxies. Checks that run before a profile holds the pack read it here.
	Proxy string `json:"proxy,omitempty"`
	// PlaceDir is the slash path below the install of the folder a profile-held loader's executable-side files are
	// placed in for a launch, when that is not MarkerDir (an Unreal game's Binaries/Win64, beside its shipping build).
	PlaceDir string `json:"placeDir,omitempty"`
	// DevTemplate is the GitHub template repo a new mod for this loader and game starts from, which the Develop tab
	// offers; nil hides the tab.
	DevTemplate *DevTemplate `json:"devTemplate,omitempty"`
}

// DevTemplate names a mod template: its GitHub repo ("owner/name") and the tag Mortar copies.
type DevTemplate struct {
	Repo string `json:"repo"`
	Ref  string `json:"ref"`
	// Commit is the full SHA the tag pointed at when the catalog was written; when set, Mortar downloads that commit
	// instead of the tag, so a moved tag cannot change what `dotnet new` and the first build run.
	Commit string `json:"commit,omitempty"`
	// Recommends are the Thunderstore packages the New mod wizard offers as starting dependencies.
	Recommends []DevRecommend `json:"recommends,omitempty"`
	// DevHelper is the Thunderstore package Develop's Test adds to the profile, for this game only, so the test copies
	// find each other; only its Package is read.
	DevHelper *DevRecommend `json:"devHelper,omitempty"`
	// AssetKinds are the mods made only of files a loader mod reads, which the New mod wizard offers beside code.
	AssetKinds []AssetKind `json:"assetKinds,omitempty"`
	// Content is the content pack: one package carrying any of its folders under mortar/, each read by its own mod.
	// Builds before it ignore the field and keep offering the AssetKinds it supersedes.
	Content *ContentKind `json:"content,omitempty"`
}

// ContentKind is the content pack a template offers, and the content folders any project of it may tick.
type ContentKind struct {
	// MinVersion is the oldest Mortar that offers it; an older build that reads the field leaves it out.
	MinVersion string          `json:"minVersion,omitempty"`
	Folders    []ContentFolder `json:"folders"`
}

// ContentFolder is a folder of a package, Folder below the content root (mortar/<ID> when empty): files matching
// Patterns, read by Dependency. Files are starter files below the root, each body's {name} replaced by the package
// name, and Needs are patterns below the root that must match a file for the folder to do anything. Legacy are the
// folders, as slash paths in the package, that older packs used for the same files; its mod still reads them, and an
// AssetKind whose Folder is one of them, or is the folder itself, is superseded by the content pack.
type ContentFolder struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Folder     string   `json:"folder,omitempty"`
	Patterns   []string `json:"patterns"`
	Dependency string   `json:"dependency"`
	// Category is the Thunderstore category slug a pack with this folder is listed under; empty adds none.
	Category string `json:"category,omitempty"`
	// Recommends are packages that go well with Dependency, offered with it when the folder is ticked.
	Recommends []string          `json:"recommends,omitempty"`
	Legacy     []string          `json:"legacy,omitempty"`
	Files      map[string]string `json:"files,omitempty"`
	Needs      []string          `json:"needs,omitempty"`
}

// Path is the folder below the content root.
func (f ContentFolder) Path() string {
	if f.Folder == "" {
		return "mortar/" + f.ID
	}
	return f.Folder
}

func (c ContentKind) validate() error {
	if _, ok := semverish.Parse(c.MinVersion); c.MinVersion != "" && !ok {
		return fmt.Errorf("content has minVersion %q, not a version", c.MinVersion)
	}
	if len(c.Folders) == 0 {
		return errors.New("content lists no folder")
	}
	for _, f := range c.Folders {
		k := AssetKind{ID: f.ID, Label: f.Label, Folder: f.Path(), Patterns: f.Patterns, Dependency: f.Dependency, Recommends: f.Recommends, Files: f.Files}
		if err := k.validate(); err != nil {
			return fmt.Errorf("content folder: %w", err)
		}
		if f.Category != "" && !categorySlug.MatchString(f.Category) {
			return fmt.Errorf("content folder %q has category %q, not a category slug", f.ID, f.Category)
		}
		for _, l := range append(slices.Clone(f.Legacy), f.Needs...) {
			if l == "." || strings.Contains(l, "\\") || !fs.ValidPath(l) {
				return fmt.Errorf("content folder %q has an unsafe path %q", f.ID, l)
			}
		}
	}
	return nil
}

// AssetKind is a mod of files a loader mod reads (suits, sounds), made without code: Folder below the package holds
// files matching Patterns, Dependency is the loader mod, and Files are starter files written at creation, each body's
// {name} replaced by the package name.
type AssetKind struct {
	ID         string            `json:"id"`
	Label      string            `json:"label"`
	Folder     string            `json:"folder"`
	Patterns   []string          `json:"patterns"`
	Dependency string            `json:"dependency"`
	Recommends []string          `json:"recommends,omitempty"`
	Files      map[string]string `json:"files,omitempty"`
}

// uploadSubdomain keeps a catalog value from steering an upload token to another host.
var uploadSubdomain = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var categorySlug = regexp.MustCompile(`^[a-z0-9-]+$`)

var assetKindID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// validate refuses a kind whose paths could leave the project folder or whose packages are not Thunderstore names.
func (k AssetKind) validate() error {
	safe := func(p string) bool { return p != "." && !strings.Contains(p, "\\") && fs.ValidPath(p) }
	if !assetKindID.MatchString(k.ID) || k.Label == "" || !safe(k.Folder) || len(k.Patterns) == 0 {
		return fmt.Errorf("asset kind %q needs an id, a label, a folder and patterns", k.ID)
	}
	for _, p := range k.Patterns {
		if _, err := path.Match(p, ""); err != nil || strings.ContainsAny(p, "/\\") {
			return fmt.Errorf("asset kind %q has a bad pattern %q", k.ID, p)
		}
	}
	for _, pkg := range append([]string{k.Dependency}, k.Recommends...) {
		if !devPackage.MatchString(pkg) {
			return fmt.Errorf("asset kind %q names %q, which is not a Thunderstore package", k.ID, pkg)
		}
	}
	for f := range k.Files {
		if !safe(f) {
			return fmt.Errorf("asset kind %q has an unsafe starter file %q", k.ID, f)
		}
	}
	return nil
}

// DevRecommend is a package a new mod likely needs: always, or when the template parameter Param was answered with one
// of Values (a choice the player picked, or "true" for a switch). Optional starts it as a soft dependency.
type DevRecommend struct {
	Package  string   `json:"package"`
	Param    string   `json:"param,omitempty"`
	Values   []string `json:"values,omitempty"`
	Optional bool     `json:"optional,omitempty"`
}

var (
	devTemplateRepo = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9][A-Za-z0-9._-]*$`)
	devTemplateRef  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	devTemplateSHA  = regexp.MustCompile(`^[0-9a-f]{40}$`)
	// devPackage is a Thunderstore package's Team-Name.
	devPackage = regexp.MustCompile(`^[A-Za-z0-9_]+-[A-Za-z0-9_]+$`)
)

func devRecommends(t *DevTemplate) []DevRecommend {
	if t == nil {
		return nil
	}
	return t.Recommends
}

// LoaderPlaceDir is the folder below the install that loader id's executable-side files go in: its entry's PlaceDir,
// else MarkerDir.
func (g GameInfo) LoaderPlaceDir(id string) string {
	for _, l := range g.Loaders {
		if l.ID == id && l.PlaceDir != "" {
			return l.PlaceDir
		}
	}
	return g.MarkerDir
}

// DoorstopProxy is the Doorstop proxy the game's BepInEx pack ships: the loader entry's Proxy, else winhttp.dll.
func (g GameInfo) DoorstopProxy() string {
	for _, l := range g.Loaders {
		if strings.HasPrefix(l.ID, "bepinex") && l.Proxy != "" {
			return l.Proxy
		}
	}
	return bepinex.Proxies()[0]
}

// A LoaderRoute's Track says where a package's files go below its folder.
const (
	// TrackSubdir places them in a folder named for the package.
	TrackSubdir = "subdir"
	// TrackKeep does so too and keeps the file's whole path from the package's top, as r2modman's subdir-no-flatten.
	TrackKeep = "subdir-no-flatten"
)

// LoaderRoute is a folder of the profile root that a mod loader's packages install into, from r2modman's install
// rules. A package folder whose name is Dir's last segment, at any depth, and a package file ending in one of
// Extensions, go to Dir.
type LoaderRoute struct {
	Dir        string   `json:"dir"`
	Extensions []string `json:"extensions,omitempty"`
	// Track is TrackSubdir or TrackKeep to place a package's files below a folder named for it, else they go straight
	// into Dir.
	Track string `json:"track,omitempty"`
	// Default marks the route that takes a file no folder or extension claims.
	Default bool `json:"default,omitempty"`
}

// Validate rejects a route that could place a file outside the profile root.
func (r LoaderRoute) Validate() error {
	if !fs.ValidPath(r.Dir) || r.Dir == "." || strings.ContainsAny(r.Dir, `\:`) {
		return fmt.Errorf("unsafe route folder %q", r.Dir)
	}
	if r.Track != "" && r.Track != TrackSubdir && r.Track != TrackKeep {
		return fmt.Errorf("route %q has unknown track %q", r.Dir, r.Track)
	}
	for _, ext := range r.Extensions {
		if len(ext) < 2 || ext[0] != '.' || strings.ContainsAny(ext, `/\*?[:`) {
			return fmt.Errorf("route %q has unusable extension %q", r.Dir, ext)
		}
	}
	return nil
}

// Place is where a file of package pkg goes in the route: rest is its path below the matched folder (its name for an
// extension match) and full its path from the package's top.
func (r LoaderRoute) Place(pkg, rest, full string) string {
	switch r.Track {
	case TrackSubdir:
		return path.Join(r.Dir, pkg, rest)
	case TrackKeep:
		return path.Join(r.Dir, pkg, full)
	}
	return path.Join(r.Dir, rest)
}

// RouteByFolder is the first route whose folder is named dir (compared without case) in its last segment.
func RouteByFolder(routes []LoaderRoute, dir string) (LoaderRoute, bool) {
	for _, r := range routes {
		if strings.EqualFold(path.Base(r.Dir), dir) {
			return r, true
		}
	}
	return LoaderRoute{}, false
}

// RouteByExtension is the route with the longest extension that ends name, as r2modman prefers the longer of two
// matches (.plugin.dll over .dll).
func RouteByExtension(routes []LoaderRoute, name string) (best LoaderRoute, ok bool) {
	longest := 0
	lower := strings.ToLower(name)
	for _, r := range routes {
		for _, ext := range r.Extensions {
			if len(ext) > longest && strings.HasSuffix(lower, strings.ToLower(ext)) {
				best, longest, ok = r, len(ext), true
			}
		}
	}
	return best, ok
}

// packMeta are the files Thunderstore requires at a package's top, which never reach a profile.
var packMeta = map[string]bool{"manifest.json": true, "icon.png": true, "readme.md": true, "licence": true}

// CleanPackagePath is the slash path of relPath below package pkg's top, ok false when it climbs out, names a drive or
// pkg has a separator. It is slash-based on every OS, since catalog validation must not differ per platform.
func CleanPackagePath(relPath, pkg string) (string, bool) {
	if pkg == "" || strings.ContainsAny(pkg, `/\:`) || pkg == "." || pkg == ".." {
		return "", false
	}
	p := path.Clean(strings.ReplaceAll(relPath, `\`, "/"))
	if p == "." || strings.HasPrefix(p, "/") || p == ".." || strings.HasPrefix(p, "../") || strings.Contains(p, ":") {
		return "", false
	}
	return p, true
}

// PackageFile cleans file relPath of Thunderstore package pkg (Namespace-Name) to a slash path below the package's top.
// ok is false for a path that climbs out of it or names a drive, a package name with a separator, and package metadata.
func PackageFile(relPath, pkg string) (clean string, ok bool) {
	p, ok := CleanPackagePath(relPath, pkg)
	if !ok {
		return "", false
	}
	if !strings.Contains(p, "/") && packMeta[strings.ToLower(p)] {
		return "", false
	}
	return p, true
}

// PlaceByRules is where r2modman's install rules put file p (from PackageFile) of package pkg: the first folder on the
// path named like a route's last segment takes the folder's contents, else the file's longest matching extension
// picks the route, and the default route takes the rest. "" means no route takes it.
func PlaceByRules(p, pkg string, routes []LoaderRoute) string {
	segs := strings.Split(p, "/")
	for i, dir := range segs[:len(segs)-1] {
		if r, ok := RouteByFolder(routes, dir); ok {
			return r.Place(pkg, path.Join(segs[i+1:]...), p)
		}
	}
	name := segs[len(segs)-1]
	if r, ok := RouteByExtension(routes, name); ok {
		return r.Place(pkg, name, p)
	}
	for _, r := range routes {
		if r.Default {
			return r.Place(pkg, name, p)
		}
	}
	return ""
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

func (sp ServerSpec) validate() error {
	if sp.Clients == "" {
		return errors.New("a server names the game whose mods it runs")
	}
	if sp.ConfigFile != "" && unsafeRel(sp.ConfigFile) {
		return fmt.Errorf("server config file %q is not a local slash path", sp.ConfigFile)
	}
	seen := map[string]bool{}
	for _, a := range sp.Args {
		if a.ID == "" || (a.Flag == "") == (a.Cvar == "") || seen[a.ID] || (a.Cvar != "" && sp.ConfigFile == "") {
			return fmt.Errorf("server argument %q needs a unique id and either a flag or a cvar with a config file", a.ID)
		}
		seen[a.ID] = true
		if _, err := regexp.Compile("^(?:" + a.Pattern + ")$"); err != nil {
			return fmt.Errorf("server argument %q has a bad pattern: %w", a.ID, err)
		}
		if a.Default != "" && !a.Matches(a.Default) {
			return fmt.Errorf("server argument %q has a default its own pattern refuses", a.ID)
		}
	}
	return nil
}

// Matches reports whether v is an allowed value; an argument with no pattern allows any.
func (a ServerArg) Matches(v string) bool {
	if a.Pattern == "" {
		return true
	}
	re, err := regexp.Compile("^(?:" + a.Pattern + ")$")
	return err == nil && re.MatchString(v)
}

// LoaderMajor is the major version the game's loader id runs, or 0 when its catalog entry names none.
func (g GameInfo) LoaderMajor(id string) int {
	for _, l := range g.Loaders {
		if l.ID == id {
			return l.Major
		}
	}
	return 0
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

// IsBepInExPackage reports whether the Thunderstore package id (Namespace-Name) is a BepInEx pack: the default one or a
// BepInEx loader's own in some catalog game. Another loader's package (MelonLoader, Northstar) versions on its own.
func IsBepInExPackage(id string) bool {
	if strings.EqualFold(id, DefaultLoaderPackage) {
		return true
	}
	for _, g := range Games() {
		for _, l := range g.Loaders {
			if strings.HasPrefix(l.ID, "bepinex") && strings.EqualFold(id, cmp.Or(l.Package, DefaultLoaderPackage)) {
				return true
			}
		}
	}
	return false
}

// LoaderPackageMajor is the BepInEx major of the Thunderstore pack id (Namespace-Name): the major the catalog names for
// the games that install it, else 5.
func LoaderPackageMajor(id string) int {
	for _, g := range Games() {
		for _, l := range g.Loaders {
			if l.Major != 0 && strings.EqualFold(id, g.LoaderPackage()) {
				return l.Major
			}
		}
	}
	return 5
}

// GameSource is a site the game's mods come from. Key is the site's name for the game (Nexus domain, Thunderstore
// community) and GameID its numeric id where the site's API uses one.
type GameSource struct {
	ID     string `json:"id"`
	Key    string `json:"key,omitempty"`
	GameID int    `json:"gameId,omitempty"`
	// UploadSubdomain is the {game} in https://{game}.curseforge.com, the host CurseForge's upload API answers on; Develop's
	// Release uploads to CurseForge only for a game that names it.
	UploadSubdomain string `json:"uploadSubdomain,omitempty"`
	// Classes are the numeric class ids a search covers, for a game whose mods sit in several (CurseForge's Create a
	// Sim and Build / Buy beside Mods); it lists Key, the class the source is addressed by.
	Classes []string `json:"classes,omitempty"`
	// Topics are further GitHub topics a search covers beside Key, for a game whose mods are tagged several ways. It is
	// not in Classes because releases before 0.3.12 reject a catalog game with a non-numeric class.
	Topics []string `json:"topics,omitempty"`
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

// SourceKeys is each of the game's sources that names the game, keyed by source id.
func (g GameInfo) SourceKeys() map[string]string {
	out := map[string]string{}
	for _, src := range g.Sources {
		if src.Key != "" {
			out[src.ID] = src.Key
		}
	}
	return out
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
	if _, ok := semverish.Parse(g.MinVersion); g.MinVersion != "" && !ok {
		return fmt.Errorf("game %q has minVersion %q, not a version", g.ID, g.MinVersion)
	}
	if len(g.Loaders) == 0 {
		return fmt.Errorf("game %q needs a loader", g.ID)
	}
	if ps := g.ProfileServer; ps != nil && (ps.Exe == "" || ps.Assembly == "" || ps.DataFlag == "") {
		return fmt.Errorf("game %q has a profile server without an exe, an assembly and a data flag", g.ID)
	}
	if g.Server != nil {
		if err := g.Server.validate(); err != nil {
			return fmt.Errorf("game %q: %w", g.ID, err)
		}
	}
	if g.Enabled && len(g.Targets) == 0 {
		return fmt.Errorf("enabled game %q needs a content target", g.ID)
	}
	if g.Enabled && g.Stores == (GameStores{}) && !g.ManualFolder {
		return fmt.Errorf("enabled game %q needs a store, or manualFolder", g.ID)
	}
	for _, check := range []func() error{g.validateNames, g.validateLoaders, g.validateSettingsAndSaves, g.validateStores} {
		if err := check(); err != nil {
			return err
		}
	}
	if err := g.validateTargets(); err != nil {
		return err
	}
	sources, err := g.validateSources()
	if err != nil {
		return err
	}
	return g.validateTemplates(sources)
}

func (g GameInfo) validateNames() error {
	names := []string{g.Marker}
	if g.LinuxMarker != "" {
		names = append(names, g.LinuxMarker)
	}
	names = append(names, g.Processes...)
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
	if g.MarkerDir != "" && unsafeRel(g.MarkerDir) {
		return fmt.Errorf("game %q has an unsafe marker folder %q", g.ID, g.MarkerDir)
	}
	for _, p := range []string{g.LinuxStart, g.DarwinMarker, g.Art} {
		if p != "" && unsafeRel(p) {
			return fmt.Errorf("game %q has an unsafe install file %q", g.ID, p)
		}
	}
	if f := g.Stores.Flatpak; f != nil && (!PlainFlatpakApp(f.App) || unsafeRel(f.Folder)) {
		return fmt.Errorf("game %q has an unusable Flatpak app %q or folder %q", g.ID, f.App, f.Folder)
	}
	if st := g.Stores.Standalone; st != nil {
		if len(st.Folders) == 0 {
			return fmt.Errorf("game %q names a standalone install without folders", g.ID)
		}
		for _, t := range st.Folders {
			for _, p := range []string{t.Windows, t.Linux, t.Darwin} {
				if p != "" && !strings.HasPrefix(p, "{") {
					return fmt.Errorf("game %q standalone folder %q must start with a token", g.ID, p)
				}
			}
		}
	}
	return nil
}

func (g GameInfo) validateLoaders() error {
	for _, l := range g.Loaders {
		if err := g.validateLoader(l); err != nil {
			return err
		}
	}
	return nil
}

func (g GameInfo) validateLoader(l GameLoader) error {
	if l.Proxy != "" && !slices.Contains(bepinex.Proxies(), l.Proxy) {
		return fmt.Errorf("game %q names an unknown Doorstop proxy %q", g.ID, l.Proxy)
	}
	if t := l.DevTemplate; t != nil && (!devTemplateRepo.MatchString(t.Repo) || !devTemplateRef.MatchString(t.Ref) || strings.Contains(t.Ref, "..")) {
		return fmt.Errorf("game %q has a mod template %q at %q that is not a GitHub repo and tag", g.ID, t.Repo, t.Ref)
	}
	if t := l.DevTemplate; t != nil && t.Commit != "" && !devTemplateSHA.MatchString(t.Commit) {
		return fmt.Errorf("game %q pins mod template %q to %q, which is not a full lowercase commit SHA", g.ID, t.Repo, t.Commit)
	}
	if t := l.DevTemplate; t != nil && t.DevHelper != nil && !devPackage.MatchString(t.DevHelper.Package) {
		return fmt.Errorf("game %q names dev helper %q, which is not a Thunderstore package", g.ID, t.DevHelper.Package)
	}
	for _, r := range devRecommends(l.DevTemplate) {
		if !devPackage.MatchString(r.Package) {
			return fmt.Errorf("game %q recommends %q, which is not a Thunderstore package", g.ID, r.Package)
		}
	}
	if t := l.DevTemplate; t != nil {
		for _, k := range t.AssetKinds {
			if err := k.validate(); err != nil {
				return fmt.Errorf("game %q: %w", g.ID, err)
			}
		}
		if t.Content != nil {
			if err := t.Content.validate(); err != nil {
				return fmt.Errorf("game %q: %w", g.ID, err)
			}
		}
	}
	if l.PlaceDir != "" && unsafeRel(l.PlaceDir) {
		return fmt.Errorf("game %q has an unsafe loader folder %q", g.ID, l.PlaceDir)
	}
	for _, r := range l.Routes {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("game %q: %w", g.ID, err)
		}
	}

	return nil
}

func (g GameInfo) validateSettingsAndSaves() error {
	for _, r := range g.RequiredSettings {
		if _, ok := g.Paths[r.Path]; !ok || r.Key == "" || r.Value == "" || r.Message == "" {
			return fmt.Errorf("game %q has a required setting without a path role of the game, a key, a value or a message", g.ID)
		}
	}
	if g.Graphics != nil {
		if err := g.Graphics.Validate(); err != nil {
			return fmt.Errorf("game %q: %w", g.ID, err)
		}
	}
	return g.validateSaves()
}

func (g GameInfo) validateSaves() error {
	for _, p := range g.SaveFiles {
		if !saveFilePattern(p) {
			return fmt.Errorf("game %q has an unusable save file pattern %q", g.ID, p)
		}
	}
	if g.SaveUserFolders && (len(g.SaveFiles) == 0 || slices.ContainsFunc(g.SaveFiles, func(p string) bool { return strings.Contains(p, "/") })) {
		return fmt.Errorf("game %q keeps saves in user folders, so its save file patterns must name files in one", g.ID)
	}
	for _, ext := range g.SaveCompanions {
		if len(ext) < 2 || ext[0] != '.' || strings.ContainsAny(ext, "/\\*?[") {
			return fmt.Errorf("game %q has an unusable save companion %q", g.ID, ext)
		}
	}
	for _, p := range g.SaveCompanionPaths {
		if !strings.Contains(p, "{stem}") || strings.Contains(p, "\\") || !fs.ValidPath(strings.ReplaceAll(p, "{stem}", "x")) {
			return fmt.Errorf("game %q has an unusable save companion path %q", g.ID, p)
		}
	}
	for _, p := range g.SaveCopies {
		if !strings.Contains(p, "{") || strings.ContainsAny(p, "/\\") {
			return fmt.Errorf("game %q has an unusable save copy pattern %q", g.ID, p)
		}
	}
	for _, p := range g.SaveLinked {
		if !strings.Contains(p, "{uuid}") || strings.Contains(p, "\\") || !fs.ValidPath(strings.ReplaceAll(p, "{uuid}", "x")) {
			return fmt.Errorf("game %q has an unusable linked save file %q", g.ID, p)
		}
	}
	for _, f := range g.SaveCopyFolders {
		if f == "" || f == "." || f == ".." || strings.ContainsAny(f, "/\\*?[") {
			return fmt.Errorf("game %q has an unusable save copy folder %q", g.ID, f)
		}
	}
	return nil
}

func (g GameInfo) validateStores() error {
	if g.Stores.Epic != nil {
		if len(g.Stores.Epic.AppNames) == 0 {
			return fmt.Errorf("game %q names Epic Games without an AppName", g.ID)
		}
		for _, app := range g.Stores.Epic.AppNames {
			if !PlainAppName(app) {
				return fmt.Errorf("game %q has an unusable Epic AppName %q", g.ID, app)
			}
		}
	}
	if g.Stores.Xbox != nil && !PlainPackage(g.Stores.Xbox.Package) {
		return fmt.Errorf("game %q has an unusable Xbox package %q", g.ID, g.Stores.Xbox.Package)
	}
	return g.validatePaths()
}

func (g GameInfo) validatePaths() error {
	for name, alts := range g.FolderNames {
		for _, n := range append([]string{name}, alts...) {
			if n == "" || n == "." || n == ".." || strings.ContainsAny(n, "/\\") {
				return fmt.Errorf("game %q has an unusable folder name %q", g.ID, n)
			}
		}
	}
	for _, cp := range g.Caches {
		if _, ok := g.Paths[cp.Role]; !ok {
			return fmt.Errorf("game %q cache %q names path role %q with no such path", g.ID, cp.Path, cp.Role)
		}
		if cp.Path == "" || unsafeRel(cp.Path) {
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
	return nil
}

func (g GameInfo) validateTargets() error {
	targets := make(map[string]struct{}, len(g.Targets))
	for _, t := range g.Targets {
		if err := g.validateTarget(t, targets); err != nil {
			return err
		}
		targets[t.ID] = struct{}{}
	}
	return nil
}

func (g GameInfo) validateTarget(t TargetDef, targets map[string]struct{}) error {
	if t.Role != "" {
		if _, ok := g.Paths[t.Role]; !ok || t.Root != "" {
			return fmt.Errorf("game %q target %q names path role %q with no such path, or also a root", g.ID, t.ID, t.Role)
		}
	} else if t.ID == "" || !underToken(t.Root, "{profileMods}", "{profile}") {
		return fmt.Errorf("game %q has a target without an id or with an unknown root %q", g.ID, t.Root)
	}
	if len(t.Sources) > 0 && t.Role == "" {
		return fmt.Errorf("game %q target %q lists sources but names no path role", g.ID, t.ID)
	}
	for _, src := range t.Sources {
		if src == "" || src != strings.ToLower(src) || strings.ContainsAny(src, " /\\") {
			return fmt.Errorf("game %q target %q has an unusable source %q", g.ID, t.ID, src)
		}
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
	return nil
}

func (g GameInfo) validateSources() (map[string]struct{}, error) {
	sources := make(map[string]struct{}, len(g.Sources))
	for _, s := range g.Sources {
		if s.ID == "" {
			return nil, fmt.Errorf("game %q has a source without an id", g.ID)
		}
		if _, ok := sources[s.ID]; ok {
			return nil, fmt.Errorf("game %q lists source %q more than once", g.ID, s.ID)
		}
		sources[s.ID] = struct{}{}
		if len(s.Classes) > 0 && !slices.Contains(s.Classes, s.Key) {
			return nil, fmt.Errorf("game %q source %q lists classes without its key %q", g.ID, s.ID, s.Key)
		}
		if s.UploadSubdomain != "" && !uploadSubdomain.MatchString(s.UploadSubdomain) {
			return nil, fmt.Errorf("game %q source %q has an upload subdomain %q that is not a host label", g.ID, s.ID, s.UploadSubdomain)
		}
		for _, c := range s.Classes {
			if _, err := strconv.ParseUint(c, 10, 31); err != nil {
				return nil, fmt.Errorf("game %q source %q has a non-numeric class %q", g.ID, s.ID, c)
			}
		}
	}
	return sources, nil
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
	// Authors are people known by different names on different mod sites.
	Authors []AuthorAliases `json:"authors,omitempty"`
	// Pins are the SHA-256 of release files the catalog vouches for beyond each component's own version: older
	// loader versions a player may choose, and the Thunderstore packs of BepInEx. An older Mortar ignores the field.
	Pins []Pin `json:"pins,omitempty"`
	// NeedsNewer lists the games Decode left out because their minVersion is newer than this Mortar.
	NeedsNewer []string `json:"-"`
}

// PinFile is the committed list of release checksums cmd/components -refresh-pins writes and every signing merges
// into the manifest's Pins.
type PinFile struct {
	Pins []Pin `json:"pins"`
}

// Pin kinds.
const (
	// PinComponent pins a version of a component (Game, Name) other than the one the manifest lists for it.
	PinComponent = "component"
	// PinPack pins a Thunderstore package version; Name is its Namespace-Name and Game is empty.
	PinPack = "pack"
)

// Pin is the SHA-256 of one release file of a loader.
type Pin struct {
	Kind    string `json:"kind"`
	Game    string `json:"game,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

// Validate checks a pin names a release and carries a SHA-256.
func (p Pin) Validate() error {
	switch p.Kind {
	case PinComponent:
		if p.Game == "" {
			return errors.New("a component pin needs a game")
		}
	case PinPack:
		if p.Game != "" {
			return errors.New("a pack pin names no game")
		}
	default:
		return fmt.Errorf("pin has unknown kind %q", p.Kind)
	}
	if p.Name == "" || p.Version == "" {
		return errors.New("a pin needs a name and a version")
	}
	if len(p.SHA256) != sha256.Size*2 {
		return fmt.Errorf("pin %s %s has an invalid sha256", p.Name, p.Version)
	}
	if _, err := hex.DecodeString(p.SHA256); err != nil {
		return fmt.Errorf("pin %s %s has an invalid sha256: %w", p.Name, p.Version, err)
	}
	return nil
}

// pinned is the SHA-256 the manifest vouches for, from a pin or, for a component, its own listed version.
func (m Manifest) pinned(kind, game, name, version string) (string, bool) {
	for _, p := range m.Pins {
		if p.Kind == kind && p.Game == game && strings.EqualFold(p.Name, name) && p.Version == version {
			return p.SHA256, true
		}
	}
	if kind == PinComponent {
		if c, ok := findComponent(m.Components, game, name); ok && c.Version == version {
			return c.SHA256, true
		}
	}
	return "", false
}

// Pin is the SHA-256 the catalog vouches for the release file of a component (kind PinComponent, game and name) or a
// Thunderstore package (kind PinPack, name Namespace-Name) at version. A Mortar that has not loaded a catalog answers
// from the bundled one.
func (c *Client) Pin(kind, game, name, version string) (string, bool) {
	if sum, ok := c.manifest.pinned(kind, game, name, version); ok {
		return sum, true
	}
	if m, err := BundledManifest(); err == nil {
		return m.pinned(kind, game, name, version)
	}
	return "", false
}

// HasPins reports whether the catalog pins any version of the component or package, which makes every unpinned
// version of it unavailable until a later catalog pins it.
func (c *Client) HasPins(kind, game, name string) bool {
	has := func(m Manifest) bool {
		return slices.ContainsFunc(m.Pins, func(p Pin) bool {
			return p.Kind == kind && p.Game == game && strings.EqualFold(p.Name, name)
		})
	}
	if has(c.manifest) {
		return true
	}
	m, err := BundledManifest()
	return err == nil && has(m)
}

// StaleAfter is how old a catalog may get before Mortar tells the player it could not refresh it.
const StaleAfter = 14 * 24 * time.Hour

// IssuedAt is when the catalog was signed: the generator makes the serial the signing time in Unix seconds (a serial
// that would not rise is bumped by one), so the signature already covers it and no older Mortar meets a new field.
func (m Manifest) IssuedAt() time.Time {
	seconds := int64(0)
	if m.Serial <= math.MaxInt64 {
		seconds = int64(m.Serial)
	}
	return time.Unix(seconds, 0).UTC()
}

// Stale reports a catalog signed more than StaleAfter before now. The copy stays in use; this only says it is not new.
func (m Manifest) Stale(now time.Time) bool {
	return m.Serial > 0 && now.Sub(m.IssuedAt()) > StaleAfter
}

// AuthorAliases is one person's names on the mod sites (a Thunderstore team and a GitHub account, say), which Browse
// counts as one author when it folds the same mod from two sites into one card.
type AuthorAliases struct {
	Names []string `json:"names"`
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
	if !validPart(c.Tag) || !validPart(c.Asset) || c.Tag == ".." || c.Asset == ".." {
		return fmt.Errorf("component %q has a release tag or asset name that is not a plain name", c.Name)
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
	for _, p := range m.Pins {
		if err := p.Validate(); err != nil {
			return err
		}
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

// Decode parses and validates a manifest after its signature has been checked. An entry this build cannot read (a
// field value a later Mortar added) is left out with a warning rather than failing the whole manifest, so a catalog
// published for newer versions still reaches this one. The generator validates strictly before signing.
func Decode(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("decode component manifest: %w", err)
	}
	m, dropped := withoutUnreadable(m)
	if len(m.Games) == 0 {
		if dropped == 0 {
			return Manifest{}, errors.New("component manifest lists no game")
		}
		return Manifest{}, errors.New("component manifest lists no game this Mortar can read")
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	enableForSandbox(m.Games)
	return m, nil
}

// withoutUnreadable is m less the components and games that fail validation, and how many games it left out.
// running is this Mortar's version (SetRunning), against which a game's MinVersion is checked; empty in tests and
// tools, which keep every game.
var running string

// SetRunning records this Mortar's version, before the first manifest is decoded.
func SetRunning(version string) { running = version }

func withoutUnreadable(m Manifest) (Manifest, int) {
	m.Components = slices.DeleteFunc(m.Components, func(c Component) bool {
		err := c.Validate()
		if err != nil {
			slog.Warn("component manifest: skipping an entry this Mortar cannot read", "game", c.Game, "component", c.Name, "err", err)
		}
		return err != nil
	})
	m.Pins = slices.DeleteFunc(m.Pins, func(p Pin) bool {
		err := p.Validate()
		if err != nil {
			slog.Warn("component manifest: skipping a pin this Mortar cannot read", "name", p.Name, "err", err)
		}
		return err != nil
	})
	listed := len(m.Games)
	m.Games = slices.DeleteFunc(m.Games, func(g GameInfo) bool {
		if c, ok := semverish.Compare(running, g.MinVersion); g.MinVersion != "" && ok && c < 0 {
			slog.Info("component manifest: leaving out a game that needs a newer Mortar", "game", g.ID, "needs", g.MinVersion)
			m.NeedsNewer = append(m.NeedsNewer, g.ID)
			return true
		}
		err := g.Validate()
		if err != nil {
			slog.Warn("component manifest: skipping a game this Mortar cannot read", "game", g.ID, "err", err)
		}
		return err != nil
	})
	for i, g := range m.Games {
		for j, l := range g.Loaders {
			if t := l.DevTemplate; t != nil && t.Content != nil {
				if c, ok := semverish.Compare(running, t.Content.MinVersion); t.Content.MinVersion != "" && ok && c < 0 {
					// A copy, so a manifest the caller still holds keeps its content pack.
					without := *t
					without.Content = nil
					m.Games[i].Loaders = slices.Clone(m.Games[i].Loaders)
					m.Games[i].Loaders[j].DevTemplate = &without
				}
			}
		}
	}
	return m, listed - len(m.Games)
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
	// KnownBrokenURL, when set, replaces the folder known-broken lists are fetched from (KnownBrokenURL).
	KnownBrokenURL string
	manifest       Manifest
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

// manifestURLs are the addresses of the published catalogs, newest release first. A release created later than a
// good one can carry a manifest that does not verify, so Load tries them in turn instead of trusting the first.
func (c *Client) manifestURLs(ctx context.Context) ([]string, error) {
	if c.ManifestURL != "" {
		return []string{c.ManifestURL}, nil
	}
	releases, err := gh.FetchReleases(ctx, c.httpClient(), cmp.Or(c.ReleasesURL, ReleasesURL))
	if err != nil {
		return nil, fmt.Errorf("component manifest response: %w", err)
	}
	var urls []string
	for _, r := range releases {
		if r.Draft || !strings.HasPrefix(r.Tag, tagPrefix) {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == "components.json" {
				urls = append(urls, a.URL)
				break
			}
		}
	}
	if len(urls) == 0 {
		return nil, errors.New("no components release is published")
	}
	return urls, nil
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
	oldSerial, floor := serialFloors(cache, bundled, publicKey)
	var fetchFailure error
	fetch := func() (cachedManifest, error) {
		_, manifest, signature, err := c.fetch(ctx, func(url string, manifest, signature []byte) error {
			if err := Verify(manifest, signature, publicKey); err != nil {
				return fmt.Errorf("%w: %s: %w", ErrBadSignature, url, err)
			}
			parsed, err := Decode(manifest)
			if err != nil {
				return err
			}
			if parsed.Serial < oldSerial {
				return fmt.Errorf("%w: %s has serial %d, below %d", ErrRollback, url, parsed.Serial, oldSerial)
			}
			if cache != nil && parsed.Serial > floor {
				_ = diskcache.Put(cache.Cache(), floorName, parsed.Serial)
			}
			return nil
		})
		if err != nil {
			fetchFailure = err
			return cachedManifest{}, err
		}
		return cachedManifest{Manifest: manifest, Signature: signature}, nil
	}

	var entry cachedManifest
	var err error
	if cache != nil {
		entry, err = diskcache.CachedStamped(cache.Cache(), cacheName, appversion.Build(), day, fetch)
	} else {
		entry, err = fetch()
	}
	if err == nil {
		var parsed Manifest
		if parsed, err = checkCached(entry, bundled, floor, publicKey); err == nil {
			c.SetManifest(parsed)
			return parsed, fetchFailure
		}
	}
	c.SetManifest(bundled)
	return bundled, err
}

// serialFloors is the serial a fetched catalog must reach: the bundled one, the highest ever accepted (floor) and the
// cached copy's, when it verifies.
func serialFloors(cache *meta.Client, bundled Manifest, publicKey []byte) (oldSerial, floor uint64) {
	oldSerial = bundled.Serial
	if cache == nil {
		return oldSerial, 0
	}
	if stored, ok := diskcache.Peek[uint64](cache.Cache(), floorName); ok {
		floor = stored.Value
		oldSerial = max(oldSerial, floor)
	}
	if old, ok := diskcache.Peek[cachedManifest](cache.Cache(), cacheName); ok {
		if err := Verify(old.Value.Manifest, old.Value.Signature, publicKey); err == nil {
			if parsed, err := Decode(old.Value.Manifest); err == nil {
				oldSerial = max(oldSerial, parsed.Serial)
			}
		}
	}
	return oldSerial, floor
}

// checkCached verifies a cache entry. The cache answers a failed fetch with its last copy however old, so its serial is
// checked here too: a blocked download must not bring back a catalog older than this build's.
func checkCached(entry cachedManifest, bundled Manifest, floor uint64, publicKey []byte) (Manifest, error) {
	if err := Verify(entry.Manifest, entry.Signature, publicKey); err != nil {
		return Manifest{}, fmt.Errorf("%w: cached copy: %w", ErrBadSignature, err)
	}
	parsed, err := Decode(entry.Manifest)
	if err != nil {
		return Manifest{}, err
	}
	if parsed.Serial < max(bundled.Serial, floor) {
		return Manifest{}, fmt.Errorf("%w: cached copy has serial %d, below the bundled %d", ErrRollback, parsed.Serial, bundled.Serial)
	}
	return parsed, nil
}

// UseCached selects the manifest an earlier run cached when it verifies and is no older than the bundled one, so
// startup reads the catalog from disk and Load's network check can run behind the open window. It reports whether
// it selected one; otherwise the bundled manifest answers until Load.
func (c *Client) UseCached(cache *meta.Client, publicKey []byte) bool {
	bundled, err := BundledManifest()
	if err != nil {
		return false
	}
	cached, ok := diskcache.Peek[cachedManifest](cache.Cache(), cacheName)
	entry := cached.Value
	if !ok || Verify(entry.Manifest, entry.Signature, publicKey) != nil {
		return false
	}
	parsed, err := Decode(entry.Manifest)
	if err != nil || parsed.Serial < bundled.Serial {
		return false
	}
	c.SetManifest(parsed)
	return true
}

// fetch returns the address, manifest and signature of the newest published catalog that accept takes. A release whose
// manifest fails accept (a bad signature, a serial below the floor) is skipped for the next older one, so one bad
// release does not block every refresh; with none accepted the newest one's failure is returned.
func (c *Client) fetch(ctx context.Context, accept func(url string, manifest, signature []byte) error) (string, []byte, []byte, error) {
	urls, err := c.manifestURLs(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	var first error
	var firstURL string
	for _, url := range urls {
		manifest, err := c.get(ctx, url, maxManifest)
		var signature []byte
		if err == nil {
			signature, err = c.get(ctx, url+".sig", maxSignature)
		}
		if err == nil {
			if err = accept(url, manifest, signature); err == nil {
				return url, manifest, signature, nil
			}
		}
		if first == nil {
			first, firstURL = err, url
		}
		if ctx.Err() != nil {
			break
		}
	}
	return firstURL, nil, nil, first
}

func (c *Client) get(ctx context.Context, address string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	body, err := httpkit.DoCapped(c.httpClient(), req, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("component manifest response: %w", err)
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
		_ = safefs.Remove(tmpPath)
		return err
	}
	defer func() {
		_ = safefs.Remove(tmpPath)
		_ = safefs.Remove(download.Sidecar(tmpPath))
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := github.Download(ctx, c.HTTP, address, tmpPath, maxComponent, "sha256:"+strings.ToLower(component.SHA256), nil); err != nil {
		return fmt.Errorf("download component %s: %w", component.Name, err)
	}
	return safefs.WriteStreamAtomic(dest, 0o600, func(w io.Writer) error {
		f, err := safefs.Open(tmpPath)
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

// unsafeRel reports a path that is not a local slash path.
func unsafeRel(p string) bool {
	return strings.Contains(p, "\\") || !fs.ValidPath(p) || p == "."
}
