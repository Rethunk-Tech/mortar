// Package bepisloader is the driver of BepisLoader, which starts BepInEx on the .NET host of Resonite: hostfxr.dll beside
// the game hands the start to BepisLoader.dll, which loads BepInEx from the folder named on the command line. Installing
// and placing the pack are thunderpack's, a mod's files and the log are BepInEx's.
package bepisloader

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-bepinex"
	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/loader/dotnetstack"
	"github.com/Rethunk-Tech/mortar/internal/loader/thunderpack"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// ID is the catalog's id for this loader.
const ID = bepinex5.BepisLoaderID

// shim is the pack's dependency, and the plugin folder its files get: without its patches the game does not run with
// BepInEx loaded.
const shim = "ResoniteModding-BepInExResoniteShim"

// beside are the pack's files the game folder must hold for a launch. Copies, as thunderpack places them: under Proton
// LinuxBootstrap.sh rewrites BepisLoader.runtimeconfig.json in place. The list is closed because whatever a pack names
// here replaces a file of the game's, so a pack's Resonite.exe stays out.
var beside = []string{"hostfxr.dll", "hookfxr.ini", "LinuxBootstrap.sh", "BepisLoader.dll", "BepisLoader.deps.json", "BepisLoader.runtimeconfig.json"}

// vanillaArgs turn off hookfxr, and Doorstop for the Unity renderer, while the files sit in the game folder.
var vanillaArgs = []string{"--hookfxr-disable", "--doorstop-enabled", "false"}

// notMods are the root namespaces of the Resonite engine, the .NET runtime and BepInEx.
var notMods = dotnetstack.Set("system", "microsoft", "mono", "mscorlib", "harmonylib", "harmonyx", "monomod", "dmd", "cecil",
	"newtonsoft", "steamworks", "bepinex", "bepisloader", "bepinexresoniteshim", "frooxengine", "elements", "protoflux",
	"skyfrost", "renderite", "awwdio", "unityengine", "unity")

var _ = loader.Register(New())

// Loader is BepisLoader as a thunderpack loader; its Index and HTTP let tests point the network elsewhere.
type Loader struct{ thunderpack.Loader }

// New is the loader with its pack rules filled in.
func New() Loader {
	spec := thunderpack.Spec{
		LoaderID: ID, Name: "BepisLoader", Root: "BepInExPack",
		Placed:        func(rel string) bool { return slices.Contains(beside, rel) },
		Map:           packPath,
		Seed:          []string{"BepInEx/config/*"},
		Files:         append([]string{"BepInEx", "Renderer", ".mortar-bepinex.json"}, beside...),
		Args:          launchArgs,
		VanillaArgs:   vanillaArgs,
		ConfigFolders: []string{"BepInEx/config", "Renderer/BepInEx/config"},
	}
	return Loader{thunderpack.Loader{Spec: spec}}
}

// packPath keeps the pack's BepInEx folder and the files that go beside the game, and drops the rest.
func packPath(rel string) string {
	if slices.Contains(beside, rel) || strings.HasPrefix(rel, "BepInEx/") {
		return rel
	}
	return ""
}

// bepinexRoot is the folder --bepinex-target names. BepisLoader runs natively under Proton, on the bootstrap's own
// dotnet, so it never takes Wine's Z: form (Gale issue 627).
func bepinexRoot(dir string) string { return bepinex.HostPath(dir, "BepInEx", false) }

// launchArgs start BepisLoader on the profile and, when a mod brought BepInExRenderer, Doorstop in the Unity renderer,
// which does run under Wine.
func launchArgs(dir string, proton bool) []string {
	args := []string{"--hookfxr-enable", "--bepinex-target", bepinexRoot(dir)}
	renderer := filepath.Join(dir, "Renderer")
	entry := bepinex.Entry(renderer)
	if !safefs.IsFile(filepath.Join(renderer, entry)) {
		return args
	}
	return append(args, "--doorstop-enabled", "true", "--doorstop-target-assembly", bepinex.HostPath(renderer, entry, proton))
}

// Install replaces the pack's own files in BepInEx/core, leaving the folders mods keep there, so a file the new release
// no longer ships does not stay. A pack without its Linux bootstrap fails here.
func (l Loader) Install(ctx context.Context, t loader.Target, pkg loader.Package, progress func(loader.Step)) (string, error) {
	core := filepath.Join(t.ProfileDir, "BepInEx", "core")
	ents, err := os.ReadDir(core)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	for _, e := range ents {
		if !e.IsDir() {
			if err := safefs.RemoveAll(filepath.Join(core, e.Name())); err != nil {
				return "", err
			}
		}
	}
	version, err := l.Loader.Install(ctx, t, pkg, progress)
	if err != nil {
		return "", err
	}
	// Extraction drops the zip's modes, and the game's Renderite.Boot runs the placed copy, which keeps this one's.
	return version, safefs.Chmod(filepath.Join(t.ProfileDir, "LinuxBootstrap.sh"), 0o755)
}

// Contribute starts the game on the profile once the shim is there. It comes in with any mod that depends on the pack,
// so a profile without mods has none and starts unmodded, as it does in r2modman and Gale, where it has no pack either.
func (l Loader) Contribute(ctx context.Context, plan *launchplan.Plan, p loader.ProfileView) error {
	if _, err := os.Stat(filepath.Join(p.Dir, "BepInEx", "plugins", shim)); err == nil {
		return l.Loader.Contribute(ctx, plan, p)
	}
	mods, _ := os.ReadDir(filepath.Join(p.Dir, "BepInEx", "plugins"))
	if len(mods) > 0 {
		return usererr.Wrap(usererr.Invalid, errors.New("this profile's mods need BepInExResoniteShim: add it from Thunderstore or enable it"))
	}
	// The target is ignored with hookfxr off; it tells Owns the profile.
	plan.AddArgs(append(slices.Clone(vanillaArgs), "--bepinex-target", bepinexRoot(p.Dir))...)
	return nil
}

// Owns is true for a game process whose BepInEx folder is this profile's.
func (Loader) Owns(p loader.Process, prof loader.ProfileView) bool {
	for i, a := range p.Args[:max(len(p.Args)-1, 0)] {
		if a == "--bepinex-target" && p.Args[i+1] == bepinexRoot(prof.Dir) {
			return true
		}
	}
	return false
}

// Formats are BepInEx's, like everything below: BepisLoader only starts it, on the same BepInEx.Core and log.
func (Loader) Formats() []string { return bepinex5.Loader{}.Formats() }

func (Loader) ModIDName(g string) string { return bepinex5.Loader{}.ModIDName(g) }

func (Loader) Path(p loader.ProfileView) (string, error) { return bepinex5.Loader{}.Path(p) }

func (Loader) Ready(line string) bool { return bepinex5.Loader{}.Ready(line) }

func (Loader) Load(line string) (string, int) { return bepinex5.Loader{}.Load(line) }

func (Loader) Analyzers() []loader.Analyzer { return bepinex5.Loader{}.Analyzers() }

func (Loader) PasteSite() string { return bepinex5.Loader{}.PasteSite() }

func (Loader) Order(p loader.ProfileView) ([]loadorder.Row, error) { return bepinex5.Loader{}.Order(p) }

func (Loader) Prelaunch(p loader.ProfileView) error { return bepinex5.Loader{}.Prelaunch(p) }

func (Loader) LaunchSettings(dir string) ([]loader.LaunchSetting, error) {
	return bepinex5.Loader{}.LaunchSettings(dir)
}

func (Loader) SetLaunchSetting(dir, id, value string) error {
	return bepinex5.Loader{}.SetLaunchSetting(dir, id, value)
}

// StackBlame names the plugin frame nearest the throw of each exception stack in BepInEx's log.
func (Loader) StackBlame(log string) []loader.Blame { return dotnetstack.Blames(log, notMods) }
