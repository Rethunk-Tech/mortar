package bepinex5

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// ID is the catalog's id for this loader.
const ID = "bepinex5"

var _ = loader.Register(Loader{})

// markerFile records what Mortar installed, since BepInEx's own files do not say which pack they came from.
const markerFile = ".mortar-bepinex.json"

// Loader is the BepInEx 5 loader. The zero value is the real loader; the fields let tests point the network elsewhere.
type Loader struct {
	// Index is the Thunderstore driver the pack is looked up in; nil means the registered one.
	Index *thunderstore.Driver
	// HTTP downloads the pack.
	HTTP *http.Client
}

type marker struct {
	Version  string `json:"version"`
	Doorstop int    `json:"doorstop"`
	// Console is the user's choice to show BepInEx's own console window; Mortar's Console tab shows the log either way.
	Console bool `json:"console,omitempty"`
}

func writeMarker(profileDir string, m marker) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return datadir.WriteFile(filepath.Join(profileDir, markerFile), b, 0o600)
}

func readMarker(profileDir string) marker {
	var m marker
	b, err := fsx.ReadFile(filepath.Join(profileDir, markerFile))
	if err != nil || json.Unmarshal(b, &m) != nil {
		return marker{Doorstop: defaultDoorstop}
	}
	return m
}

func (Loader) ID() string { return ID }

// ImportRoots is BepInEx's folder, where an r2modman profile's loose files belong.
func (Loader) ImportRoots() []string { return []string{"BepInEx"} }

// VersionScheme is Thunderstore's strict d.d.d.
func (Loader) VersionScheme() string { return deps.SemverStrict }

func (Loader) Formats() []string { return []string{"bepinplugin"} }

func (Loader) Status(t loader.Target) (loader.Status, error) {
	if _, err := os.Stat(filepath.Join(t.ProfileDir, preloader)); errors.Is(err, fs.ErrNotExist) {
		return loader.Status{}, nil
	} else if err != nil {
		return loader.Status{}, err
	}
	m := readMarker(t.ProfileDir)
	return loader.Status{Installed: true, Version: m.Version}, nil
}

// Install lays the BepInExPack archive out in the profile and records its version and Doorstop major.
func (Loader) Install(_ context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	got, err := InstallPack(pkg.Archive, t.ProfileDir)
	if err != nil {
		return "", err
	}
	if progress != nil {
		progress(loader.StepFiles)
	}
	m := readMarker(t.ProfileDir)
	m.Version, m.Doorstop = got.Version, got.Doorstop
	if err := writeMarker(t.ProfileDir, m); err != nil {
		return "", err
	}
	return got.Version, nil
}

// Contribute points Doorstop at the profile's preloader, declares the proxy files for the game folder and, under
// Proton, asks for the winhttp override. A native Linux build is started through LD_PRELOAD instead (see contributeLinux).
func (Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	m := readMarker(p.Dir)
	if p.Platform == "linux" {
		return contributeLinux(plan, p, m)
	}
	proton := p.Runtime == "proton"
	plan.AddArgs(LaunchArgs(p.Dir, m.Doorstop, proton)...)
	for _, f := range DoorstopFiles(p.Dir) {
		plan.AddFile(launchplan.PlanFile{Src: filepath.Join(p.Dir, filepath.FromSlash(f)), Dst: filepath.FromSlash(f)})
	}
	if proton {
		plan.RequireRuntime(launchplan.RuntimeReq{Kind: "dll-override", Key: "winhttp", Value: "native,builtin"})
	}
	return nil
}

// contributeLinux starts a native Linux build directly with the pack's libdoorstop preloaded and the Doorstop
// settings in the environment, as the pack's run_bepinex.sh does; nothing is placed in the game folder. A store relay
// cannot carry that environment, so the start is direct. The Doorstop flags are passed too, ignored by the game,
// so Owns can tell the profile's process.
func contributeLinux(plan *launchplan.Plan, p loader.ProfileView, m marker) error {
	g, _ := components.Game(p.Game)
	if g.LinuxMarker == "" {
		return errors.New(g.Name + " names no native Linux executable")
	}
	lib, err := linuxDoorstop(p.Dir)
	if err != nil {
		return err
	}
	if err := plan.OverrideExe(ID, filepath.Join(p.InstallDir, g.LinuxMarker)); err != nil {
		return err
	}
	plan.AddArgs(LaunchArgs(p.Dir, m.Doorstop, false)...)
	target := targetPath(p.Dir, false)
	if m.Doorstop >= 4 {
		plan.SetEnv("DOORSTOP_ENABLED", "1")
		plan.SetEnv("DOORSTOP_TARGET_ASSEMBLY", target)
	} else {
		plan.SetEnv("DOORSTOP_ENABLE", "TRUE")
		plan.SetEnv("DOORSTOP_INVOKE_DLL_PATH", target)
	}
	plan.SetEnv("LD_PRELOAD", lib)
	// Steamworks reads the app id from here when Steam did not start the process, instead of restarting it through
	// Steam without the preload.
	if id := g.SteamAppID(); id != "" {
		plan.SetEnv("SteamAppId", id)
	}
	return nil
}

// linuxDoorstop is the pack's 64-bit Linux Doorstop library. Only a pack for a game with a native Linux build ships
// one (Valheim's); the shared BepInExPack is Windows-only.
func linuxDoorstop(profileDir string) (string, error) {
	p := filepath.Join(profileDir, "doorstop_libs", "libdoorstop_x64.so")
	if st, err := os.Stat(p); err != nil || !st.Mode().IsRegular() {
		return "", errors.New("the BepInEx pack has no 64-bit Linux Doorstop library; reinstall the loader")
	}
	return filepath.Abs(p)
}

// Owns is true for a game process whose Doorstop target is this profile's preloader, in either the host or the Wine
// Z: form Contribute passes it in. A game started without Mortar's flags runs no profile Mortar can name.
func (Loader) Owns(p loader.Process, prof loader.ProfileView) bool {
	for i, a := range p.Args[:max(len(p.Args)-1, 0)] {
		if a != "--doorstop-target" && a != "--doorstop-target-assembly" {
			continue
		}
		if t := p.Args[i+1]; t == targetPath(prof.Dir, false) || t == targetPath(prof.Dir, true) {
			return true
		}
	}
	return false
}

// Vanilla switches Doorstop off, so the game starts unmodded even while the proxy files sit in its folder.
func (Loader) Vanilla(_ context.Context, plan *launchplan.Plan, _ loader.Target) error {
	plan.AddArgs(VanillaArgs()...)
	return nil
}

// Prelaunch sets BepInEx's console window to the user's choice, off unless they asked for it: Mortar follows
// LogOutput.log into its own Console tab, the one channel that works under Proton too.
func (Loader) Prelaunch(p loader.ProfileView) error {
	return applyConsole(p.Dir, readMarker(p.Dir).Console)
}

// NeedsWinHTTPOverride is true: Doorstop injects through winhttp.dll, which Wine only loads when the prefix overrides it.
func (Loader) NeedsWinHTTPOverride() bool { return true }

func (Loader) Path(p loader.ProfileView) (string, error) {
	return filepath.Join(p.Dir, "BepInEx", "LogOutput.log"), nil
}

// PasteSite is paste.gg: anonymous and general-purpose with a 15 MiB limit, above a stored run's cap, where mclo.gs
// truncates past 25,000 lines and Pastebin takes 512 KiB without an account.
func (Loader) PasteSite() string { return "https://paste.gg/" }

// Ready is BepInEx's last startup line.
func (Loader) Ready(line string) bool { return strings.Contains(line, "Chainloader startup complete") }

func (Loader) Analyzers() []loader.Analyzer { return []loader.Analyzer{analyzer{}} }

type analyzer struct{}

func (analyzer) ID() string { return ID }

func (analyzer) Analyze(logs loader.Logs) []loader.Finding { return Analyze(logs.Loader, logs.Player) }

// PlayerLogRole is the catalog path of Unity's Player.log, where exceptions thrown by plugins land.
func (Loader) PlayerLogRole() string { return "unityLog" }

// ConfigDirs is where BepInEx plugins keep their .cfg files.
func (Loader) ConfigDirs() []string { return []string{"BepInEx/config"} }

// ModArchive reports a Thunderstore-shaped archive: manifest.json and icon.png at its root.
func (Loader) ModArchive(names []string) bool {
	root := func(want string) bool {
		for _, n := range names {
			if strings.EqualFold(n, want) {
				return true
			}
		}
		return false
	}
	return root("manifest.json") && root("icon.png")
}
