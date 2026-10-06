package smapi

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

// ID is the catalog's id for this loader.
const ID = "smapi"

// gameID is the only game SMAPI loads.
const gameID = "stardew"

var _ = loader.Register(Loader{})

// Loader is SMAPI. The zero value is the real loader; the fields exist so tests can point network and filesystem
// operations elsewhere.
type Loader struct {
	Client       *http.Client
	Components   *components.Client
	ReleasesURL  string
	DownloadBase string
	AssetPattern string
	// CacheDir holds the cached release lookup.
	CacheDir string
	// LogDir holds SMAPI-latest.txt.
	LogDir string
}

// gameInfo is Stardew Valley's catalog entry, which the bundled manifest always carries.
func gameInfo() components.GameInfo {
	g, _ := components.Game(gameID)
	return g
}

func (Loader) ID() string { return ID }

// InstalledVersion is the game version SMAPI's last log recorded for the install.
func (l Loader) InstalledVersion(installDir string) string {
	st, _ := l.Status(loader.Target{InstallDir: installDir})
	return st.GameVersion
}

// VersionScheme is SMAPI's lenient semver.
func (Loader) VersionScheme() string { return deps.SemverSMAPI }

func (Loader) Formats() []string { return []string{"smapi"} }

// ProcessNames is SMAPI's executable, the process a running profile is found by.
func (Loader) ProcessNames() []string { return []string{smapiMarker} }

// DeniedArgs are the SMAPI flags Mortar always sets itself.
func (Loader) DeniedArgs() []string {
	return []string{"--mods-path", "--no-terminal", "--skip-terminal"}
}

// SteamExe is the executable Steam's launch options start in place of the game.
func (Loader) SteamExe(installDir string) string {
	return filepath.Join(installDir, smapiMarker+".exe")
}

func modsDir(p loader.ProfileView) string { return filepath.Join(p.Dir, "mods") }

// Contribute points SMAPI at the profile's mods folder.
func (Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	return contribute(buildOS(p.InstallDir), plan, p)
}

// contribute: Linux SMAPI's launcher reads its own flags before `--` and forwards only what follows it.
func contribute(goos string, plan *launchplan.Plan, p loader.ProfileView) error {
	mods := modsDir(p)
	if !filepath.IsAbs(mods) {
		return fmt.Errorf("mods folder %q is not an absolute path", mods)
	}
	if goos == "windows" {
		plan.SetEntry(filepath.Join(p.InstallDir, smapiMarker+".exe"))
	} else {
		plan.AddArgs("--skip-terminal", "--")
		plan.SetEntry(filepath.Join(p.InstallDir, linuxLauncher))
	}
	plan.AddArgs("--mods-path", mods)
	plan.AddProcessName(smapiMarker)
	return nil
}

// startupMarkers are files SMAPI leaves in smapi-internal for its next start, which then waits for a key press. Mortar
// gives the game no console input, so that wait throws and SMAPI exits before the game opens, on every later start.
// The crash log itself (ErrorLogs/SMAPI-crash.txt) is kept.
var startupMarkers = []string{"StardewModdingAPI.crash.marker", "StardewModdingAPI.update.marker"}

// Prelaunch removes SMAPI's startup markers, so a crash in the last session cannot stop this one from starting.
func (Loader) Prelaunch(p loader.ProfileView) error {
	if p.InstallDir == "" {
		return nil
	}
	for _, name := range startupMarkers {
		if err := fsx.RemoveAll(filepath.Join(p.InstallDir, "smapi-internal", name)); err != nil {
			return err
		}
	}
	return nil
}

// Vanilla starts the unmodded game. SMAPI's unix-launcher.sh always execs StardewModdingAPI, so on Linux the game
// the installer kept as StardewValley-original is run directly; Windows has a plain executable.
func (Loader) Vanilla(_ context.Context, plan *launchplan.Plan, t loader.Target) error {
	return vanilla(buildOS(t.InstallDir), plan, t.InstallDir)
}

func vanilla(goos string, plan *launchplan.Plan, dir string) error {
	plan.AddProcessName(smapiMarker)
	if goos == "windows" {
		plan.SetEntry(filepath.Join(dir, "Stardew Valley.exe"))
		return nil
	}
	return plan.OverrideExe(ID, filepath.Join(dir, linuxOriginal))
}

// Owns is true for a process started with this profile's mods folder.
func (Loader) Owns(p loader.Process, prof loader.ProfileView) bool {
	return launch.Process{PID: p.PID, Args: p.Args}.UsesModsPath(modsDir(prof))
}

// Path is SMAPI-latest.txt, which SMAPI rewrites on each start and which outlives the game.
func (l Loader) Path(loader.ProfileView) (string, error) {
	dir, err := l.logDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SMAPI-latest.txt"), nil
}

// Ready is true for the line SMAPI logs once it has loaded the mods.
func (Loader) Ready(line string) bool { return strings.Contains(line, " SMAPI] Loaded ") }

// Analyzers is none: SMAPI's log problems are found by internal/launch.
func (Loader) Analyzers() []loader.Analyzer { return nil }

// Send runs command in the game through the bridge mod in the profile.
func (Loader) Send(ctx context.Context, _ loader.Target, p loader.ProfileView, command string) (string, error) {
	if p.Companion == "" {
		return "", fmt.Errorf("this profile has no console bridge")
	}
	return "", bridge.Send(ctx, filepath.Join(p.Companion, bridge.SMAPI.StateFile), command)
}

// ReportsStartup is true: the Mortar SMAPI Bridge times each mod's startup.
func (Loader) ReportsStartup() {}

// FeedsOverlay is true: the Mortar SMAPI Bridge serves the stream overlay's values.
func (Loader) FeedsOverlay() {}

// SharesLog marks SMAPI's log as the one smapi.io's parser reads.
func (Loader) SharesLog() {}

// Order is SMAPI's load order of the enabled mods, each folder in p.Enabled keeping its manifest at its root.
func (Loader) Order(p loader.ProfileView) ([]loadorder.Row, error) {
	var mods []loadorder.Mod
	for _, dir := range p.Enabled {
		b, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName))
		if err != nil {
			continue
		}
		mf, err := manifest.Parse(b)
		if err != nil {
			continue
		}
		m := loadorder.Mod{ID: mf.ModID(), Name: mf.Name, ContentPackFor: mf.ContentPackForID()}
		for _, d := range mf.Dependencies {
			if d.Required {
				m.Needs = append(m.Needs, d.ModID())
			} else {
				m.Optional = append(m.Optional, d.ModID())
			}
		}
		mods = append(mods, m)
	}
	return loadorder.Resolve(mods), nil
}

// Companion is the Mortar SMAPI Bridge.
func (Loader) Companion() bridge.Companion { return bridge.SMAPI }

// GameVersion returns the game version in a SMAPI log header ("SMAPI x with Stardew Valley y").
func (Loader) GameVersion(log string) string {
	for line := range strings.SplitSeq(log, "\n") {
		if m := logHeader.FindStringSubmatch(strings.TrimRight(line, "\r")); m != nil {
			return m[2]
		}
	}
	return ""
}

// ModArchive reports an archive holding a SMAPI mod: a manifest.json at any depth.
func (Loader) ModArchive(names []string) bool {
	for _, n := range names {
		if strings.EqualFold(path.Base(strings.ReplaceAll(n, `\`, "/")), manifest.FileName) {
			return true
		}
	}
	return false
}
