package melonloader

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/wineoverride"
)

var _ = loader.Register(Loader{})

// Loader is the MelonLoader driver. The zero value is the real loader; the fields let tests point the network elsewhere.
type Loader struct {
	// Components is the catalog the loader's release is looked up in; nil means the active one.
	Components *components.Client
	// HTTP calls GitHub and downloads releases.
	HTTP *http.Client
	// APIBase replaces https://api.github.com.
	APIBase string
}

// marker is what Mortar records in the profile: the installed version and the launch options it keeps for the profile.
type marker struct {
	Version string `json:"version"`
	// Console is the user's choice to show MelonLoader's console window; Mortar's Console tab shows the log either way.
	Console  bool   `json:"console,omitempty"`
	LogLevel string `json:"logLevel,omitempty"`
}

func readMarker(profileDir string) marker {
	var m marker
	if b, err := fsx.ReadFile(filepath.Join(profileDir, markerFile)); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	return m
}

func writeMarker(profileDir string, m marker) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return datadir.WriteFile(filepath.Join(profileDir, markerFile), b, 0o600)
}

func (Loader) ID() string { return ID }

func (Loader) Formats() []string { return []string{"melon"} }

// Status is installed when the profile holds the loader folder and its proxy.
func (Loader) Status(t loader.Target) (loader.Status, error) {
	if !fsx.IsDir(filepath.Join(t.ProfileDir, loaderDir)) || !fsx.IsFile(filepath.Join(t.ProfileDir, proxyDLL)) {
		return loader.Status{}, nil
	}
	return loader.Status{Installed: true, Version: readMarker(t.ProfileDir).Version}, nil
}

// Install lays the release zip out in the profile and records its version.
func (Loader) Install(_ context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	if err := InstallPack(pkg.Archive, t.ProfileDir); err != nil {
		return "", err
	}
	if progress != nil {
		progress(loader.StepFiles)
	}
	m := readMarker(t.ProfileDir)
	m.Version = pkg.Version
	if err := writeMarker(t.ProfileDir, m); err != nil {
		return "", err
	}
	return pkg.Version, nil
}

// baseDirFlag names the folder MelonLoader keeps its loader folder, mods and settings in.
const baseDirFlag = "--melonloader.basedir"

// LaunchArgs point MelonLoader at the profile as its base folder, and ask it to write the game's assemblies on a start
// that has none yet. proton gives the path in the Wine Z: form.
func LaunchArgs(profileRoot string, proton bool) []string {
	args := []string{baseDirFlag, basePath(profileRoot, proton)}
	if !generated(profileRoot) {
		args = append(args, "--melonloader.agfregenerate")
	}
	return args
}

func basePath(profileRoot string, proton bool) string {
	p, err := filepath.Abs(profileRoot)
	if err != nil {
		p = profileRoot
	}
	if proton {
		return wineoverride.ZPath(p)
	}
	return p
}

// Contribute starts MelonLoader on the profile and declares the proxy for the game folder. Only Windows builds run
// MelonLoader here, under Proton or on Windows, where the proxy is a library the prefix loads natively.
func (Loader) Contribute(_ context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	if p.Platform == "linux" {
		return errors.New("MelonLoader starts only a game's Windows build, which runs under Proton")
	}
	proton := p.Runtime == "proton" || p.Runtime == "mortar-proton"
	plan.AddArgs(LaunchArgs(p.Dir, proton)...)
	plan.AddArgs(settingArgs(p.Dir)...)
	plan.AddFile(launchplan.PlanFile{Src: filepath.Join(p.Dir, proxyDLL), Dst: proxyDLL})
	if proton {
		plan.RequireRuntime(launchplan.RuntimeReq{Kind: "dll-override", Key: "version", Value: "native,builtin"})
	}
	return nil
}

// Owns is true for a game process started on this profile's base folder, in either the host or the Wine Z: form.
func (Loader) Owns(p loader.Process, prof loader.ProfileView) bool {
	for i, a := range p.Args[:max(len(p.Args)-1, 0)] {
		if a == baseDirFlag && (p.Args[i+1] == basePath(prof.Dir, false) || p.Args[i+1] == basePath(prof.Dir, true)) {
			return true
		}
	}
	return false
}

// Vanilla starts the game with MelonLoader loaded but no mods, whatever proxy is left in the game folder.
func (Loader) Vanilla(_ context.Context, plan *launchplan.Plan, _ loader.Target) error {
	plan.AddArgs("--no-mods")
	return nil
}

// OverrideDLL is version: MelonLoader injects through version.dll, which Wine only loads when the prefix overrides it.
func (Loader) OverrideDLL(string) string { return "version" }

// ProfileFiles are what InstallPack lays out in each profile's folder, with the marker that records the version.
func (Loader) ProfileFiles() []string { return []string{loaderDir, proxyDLL, markerFile} }

// ConfigDirs is where MelonLoader and its mods keep their settings.
func (Loader) ConfigDirs() []string { return []string{"UserData"} }

// VersionScheme is Thunderstore's strict d.d.d.
func (Loader) VersionScheme() string { return deps.SemverStrict }

// Path is MelonLoader's log of the latest start.
func (Loader) Path(p loader.ProfileView) (string, error) {
	return filepath.Join(p.Dir, loaderDir, "Latest.log"), nil
}

// PasteSite is paste.gg, as for BepInEx's log: no parser site reads MelonLoader's.
func (Loader) PasteSite() string { return "https://paste.gg/" }

var modsLoaded = regexp.MustCompile(`\d+ Mods? loaded\.`)

// Ready is the line MelonLoader writes when it has loaded the Mods folder, its last loading step.
func (Loader) Ready(line string) bool { return modsLoaded.MatchString(line) }

// Analyzers are none: MelonLoader's log has no parser here yet.
func (Loader) Analyzers() []loader.Analyzer { return nil }

// ModArchive reports a Thunderstore-shaped archive: manifest.json and icon.png at its root.
func (Loader) ModArchive(names []string) bool {
	has := func(want string) bool {
		for _, n := range names {
			if strings.EqualFold(n, want) {
				return true
			}
		}
		return false
	}
	return has("manifest.json") && has("icon.png")
}

// PlayerLogRole is the catalog path of Unity's Player.log, where exceptions Unity caught land.
func (Loader) PlayerLogRole() string { return "unityLog" }

// StackBlame names the mod frame nearest the throw of each exception stack in MelonLoader's log or Unity's Player.log.
func (Loader) StackBlame(log string) []loader.Blame {
	return dotnetstack.Blames(log, dotnetstack.Unity)
}
