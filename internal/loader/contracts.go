package loader

import (
	"context"
	"encoding/json"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// ProfileView is the read-only part of a profile a loader needs. It is defined here so loaders do not depend on the
// profile package.
type ProfileView struct {
	Game string
	// Dir is the profile's folder; loader files live in it.
	Dir        string
	InstallDir string
	// Runtime is the id of the runtime that runs the install: native or proton.
	Runtime string
	// Enabled are the folders of the profile's enabled packages.
	Enabled []string
	// Companion is the folder of the loader's companion mod in the profile (a loader whose companion is not a mod folder
	// of its own leaves it empty).
	Companion string
}

// Target is where a loader installs: the game's install and the profile's folder.
type Target struct {
	Game       string
	InstallDir string
	ProfileDir string
	// StoreDir is a store-wide folder some loaders fill (Minecraft's versions).
	StoreDir string
	Runtime  string
	// Exec runs a program under the install's runtime, for loaders whose installer is one.
	Exec func(ctx context.Context, req launchplan.RuntimeReq, argv []string) error
	// Bundled receives the loader's own mods while an install's files still exist; nil when the loader has none.
	Bundled Bundled
}

// Package is a loader package already downloaded: its identity and the archive on disk.
type Package struct {
	ID, Version, Archive string
}

// Loader is a game's mod loader. The optional capabilities below are asked for by type assertion.
type Loader interface {
	ID() string
	// Formats are the component formats the loader loads.
	Formats() []string
	// Status reads the loader's state from disk.
	Status(t Target) (Status, error)
	// Install puts the loader into the profile and returns its version.
	Install(ctx context.Context, t Target, pkg Package, progress func(Step)) (string, error)
	// Contribute adds the loader's profile side to plan; install-side files go in plan.Files for the deployer.
	Contribute(ctx context.Context, plan *launchplan.Plan, p ProfileView) error
}

// Vanilla starts the game without the loader.
type Vanilla interface {
	Vanilla(ctx context.Context, plan *launchplan.Plan, t Target) error
}

// Process is a running game process.
type Process struct {
	PID  int
	Args []string
}

// Owner decides whether a running process belongs to this profile's launch.
type Owner interface {
	Owns(p Process, prof ProfileView) bool
}

// Finding is one load failure read from a log.
type Finding struct {
	Kind string
	// Plugin is the plugin as the log names it.
	Plugin  string
	Message string
	// Line is the 1-based line in the log the finding came from.
	Line int
	// Source is the log: LogOutput.log or Player.log.
	Source string
}

// Logs are a launch's log texts; a loader fills the ones it has.
type Logs struct{ Loader, Player string }

// Analyzer turns logs into findings.
type Analyzer interface {
	ID() string
	Analyze(logs Logs) []Finding
}

// WithLogs is a loader with a log Mortar follows and analyses.
type WithLogs interface {
	Path(p ProfileView) (string, error)
	// Ready reports whether a log line says the loader has finished loading.
	Ready(line string) bool
	Analyzers() []Analyzer
}

// InstalledVersion is a loader that can read the game's version from an install.
type InstalledVersion interface {
	InstalledVersion(installDir string) string
}

// WithPlayerLog is a loader whose analyzers also read the game's Unity player log, which the catalog places under
// this path role.
type WithPlayerLog interface{ PlayerLogRole() string }

// WithConfig is a loader whose mods keep settings files in the profile that Mortar may edit.
type WithConfig interface {
	// ConfigDirs are the writable folders, relative to the profile's folder.
	ConfigDirs() []string
}

// ComponentID names one loaded component.
type ComponentID struct{ Format, ID string }

// WithOrder is a loader with a computed load order.
type WithOrder interface {
	Order(p ProfileView) ([]ComponentID, error)
}

// StartupTimings is a loader whose companion records how long each mod adds to the game's startup.
type StartupTimings interface{ ReportsStartup() }

// OrderWriter is a loader whose load order lives in a file of the game.
type OrderWriter interface {
	WriteOrder(ctx context.Context, t Target, order []ComponentID) error
}

// WithCompanion is a loader with a companion mod that lets Mortar talk to the running game.
type WithCompanion interface {
	Companion() bridge.Companion
}

// Querier is a loader whose running game answers questions (status, plugins) through its companion.
type Querier interface {
	Query(ctx context.Context, t Target, p ProfileView, what string) (json.RawMessage, error)
}

// Console is a loader whose running game takes commands.
type Console interface {
	Send(ctx context.Context, t Target, p ProfileView, command string) (string, error)
}

// WinHTTPLoader is a loader that injects through winhttp.dll, which Wine only loads when the prefix overrides it.
type WinHTTPLoader interface{ NeedsWinHTTPOverride() bool }

// Releases is a loader whose releases Mortar looks up and downloads. g is the game's catalog entry, for a loader
// that finds its releases through the game's sources.
type Releases interface {
	// Latest is the newest stable version.
	Latest(ctx context.Context, g components.GameInfo) (string, error)
	// Versions are the recent releases, newest first.
	Versions(ctx context.Context, g components.GameInfo) ([]string, error)
	// Fetch downloads the installer package of version to dst.
	Fetch(ctx context.Context, g components.GameInfo, version, dst string) error
}

// InProfile is a loader whose files live in each profile's folder, so an install runs once per profile.
type InProfile interface{ InProfile() }

// BundledCopier is a loader that ships mods of its own, which Mortar can copy from an install the loader was put into
// outside Mortar.
type BundledCopier interface {
	CopyBundled(installDir, dst string) error
	// BundleSource is the kind and display name a profile gives the loader's bundled mods.
	BundleSource() (kind, name string)
}

// VersionScheme is a loader whose components are versioned in one scheme (deps.SemverSMAPI, deps.SemverStrict,
// deps.Opaque); a dependency's constraint is read in the owner's loader's scheme.
type VersionScheme interface{ VersionScheme() string }

// ProcessNames is a loader whose running game is found by these executable names.
type ProcessNames interface{ ProcessNames() []string }

// DeniedArgs is a loader that sets some launch flags itself, which a profile's own launch options may not repeat.
type DeniedArgs interface{ DeniedArgs() []string }

// SteamExe is a loader that Steam's launch options start in place of the game: the executable they run.
type SteamExe interface {
	SteamExe(installDir string) string
}

// GameVersion is a loader whose log says which game version it ran with.
type GameVersion interface {
	// GameVersion is the version a log's text names, "" when it names none.
	GameVersion(log string) string
}
