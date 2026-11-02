package bepisloader

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-bepinex/wineoverride"
	"github.com/Rethunk-Tech/go-safefs"

	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
)

// realPack is BepisLoader 1.7.0's file layout, core libraries shortened to the ones that name the runtime.
var realPack = []string{
	"manifest.json", "icon.png", "README.md", "LICENSE",
	"BepInExPack/BepisLoader.runtimeconfig.json", "BepInExPack/hookfxr.ini", "BepInExPack/LinuxBootstrap.sh",
	"BepInExPack/BepisLoader.dll", "BepInExPack/BepisLoader.deps.json", "BepInExPack/hostfxr.dll", "BepInExPack/BepInEx/icon.ico",
	"BepInExPack/BepInEx/core/BepInEx.NET.CoreCLR.dll", "BepInExPack/BepInEx/core/BepInEx.Core.dll", "BepInExPack/BepInEx/core/0Harmony.dll",
}

func writeZip(t *testing.T, names ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pack.zip")
	f, err := safefs.CreateExcl(p, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, n := range names {
		h := &zip.FileHeader{Name: n, Method: zip.Deflate}
		h.SetMode(0o644)
		if strings.HasSuffix(n, ".sh") {
			h.SetMode(0o755)
		}
		e, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.Write([]byte(n)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func installed(t *testing.T, names ...string) (Loader, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "profile")
	l := New()
	if _, err := l.Install(context.Background(), loader.Target{ProfileDir: dir}, loader.Package{ID: ID, Version: "1.7.0", Archive: writeZip(t, names...)}, nil); err != nil {
		t.Fatal(err)
	}
	return l, dir
}

func view(dir, rt string) loader.ProfileView {
	return loader.ProfileView{Game: "resonite", Dir: dir, Platform: "windows", Runtime: rt}
}

func TestInstallLaysOutThePackAndAnUpdateReplacesIt(t *testing.T) {
	l, dir := installed(t, realPack...)
	st, err := l.Status(loader.Target{ProfileDir: dir})
	if err != nil || !st.Installed || st.Version != "1.7.0" {
		t.Fatalf("status = %+v, %v", st, err)
	}
	for _, f := range append(slices.Clone(beside), "BepInEx/core/BepInEx.NET.CoreCLR.dll") {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err == nil {
		t.Error("the package's manifest reached the profile")
	}
	if runtime.GOOS != "windows" {
		// The bootstrap is run by the game's Renderite.Boot, so its mode must survive the extract.
		st, err := os.Stat(filepath.Join(dir, "LinuxBootstrap.sh"))
		if err != nil || st.Mode()&0o100 == 0 {
			t.Errorf("LinuxBootstrap.sh mode = %v, %v", st, err)
		}
	}

	touch(t, filepath.Join(dir, "BepInEx", "core", "Stale.dll"))
	touch(t, filepath.Join(dir, "BepInEx", "core", "Ns-Mod", "Mod.dll"))
	touch(t, filepath.Join(dir, "BepInEx", "config", "BepInEx.cfg"))
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "config", "BepInEx.cfg"), []byte("mine"), 0o600); err != nil {
		t.Fatal(err)
	}
	update := append(slices.Clone(realPack), "BepInExPack/BepInEx/config/BepInEx.cfg")
	if _, err := l.Install(context.Background(), loader.Target{ProfileDir: dir}, loader.Package{ID: ID, Version: "1.8.0", Archive: writeZip(t, update...)}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx", "core", "Stale.dll")); err == nil {
		t.Error("a file the update does not ship stayed in core")
	}
	if _, err := os.Stat(filepath.Join(dir, "BepInEx", "core", "Ns-Mod", "Mod.dll")); err != nil {
		t.Error("a mod's core folder went with the update:", err)
	}
	if b, _ := safefs.ReadFile(filepath.Join(dir, "BepInEx", "config", "BepInEx.cfg")); string(b) != "mine" {
		t.Errorf("the player's config = %q", b)
	}
}

func TestStatusWithoutThePack(t *testing.T) {
	st, err := New().Status(loader.Target{ProfileDir: t.TempDir()})
	if err != nil || st.Installed {
		t.Fatalf("status = %+v, %v", st, err)
	}
}

func TestPackFilesOffTheListStayOutOfTheGameFolder(t *testing.T) {
	pack := append(slices.Clone(realPack), "BepInExPack/Resonite.exe", "BepInExPack/Renderite.Host.dll", "BepInExPack/HOSTFXR.DLL.bak", "BepInExPack/sub/hostfxr.dll", "BepInExPack/Renderer/x.dll")
	l, dir := installed(t, pack...)
	touch(t, filepath.Join(dir, "BepInEx", "plugins", shim, "BepInExResoniteShim", "BepInExResoniteShim.dll"))
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(context.Background(), plan, view(dir, "native")); err != nil {
		t.Fatal(err)
	}
	var placed []string
	for _, f := range plan.Files {
		placed = append(placed, filepath.ToSlash(f.Dst))
		if !strings.HasPrefix(f.Src, dir) {
			t.Errorf("%s comes from outside the profile", f.Src)
		}
	}
	slices.Sort(placed)
	want := slices.Sorted(slices.Values(beside))
	if !slices.Equal(placed, want) {
		t.Errorf("placed %v, want %v", placed, want)
	}
	for _, f := range []string{"Resonite.exe", "Renderite.Host.dll", "sub", "Renderer"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			t.Errorf("%s reached the profile", f)
		}
	}
}

func TestInstallRefusesAClimbingName(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profile")
	archive := writeZip(t, "manifest.json", "BepInExPack/../../hostfxr.dll")
	if _, err := New().Install(context.Background(), loader.Target{ProfileDir: dir}, loader.Package{ID: ID, Version: "1", Archive: archive}, nil); err == nil {
		t.Fatal("a name climbing out of the pack installed")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "hostfxr.dll")); err == nil {
		t.Fatal("the file landed beside the profile")
	}
}

func TestContributeArguments(t *testing.T) {
	cases := []struct {
		name, runtime string
		renderer      bool
		doorstopWine  bool
	}{
		{name: "Windows", runtime: "native"},
		{name: "Proton", runtime: "proton"},
		{name: "Windows with the renderer", runtime: "native", renderer: true},
		{name: "Proton with the renderer: only Doorstop takes Wine's form", runtime: "proton", renderer: true, doorstopWine: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, dir := installed(t, realPack...)
			touch(t, filepath.Join(dir, "BepInEx", "plugins", shim, "BepInExResoniteShim.dll"))
			want := []string{"--hookfxr-enable", "--bepinex-target", filepath.Join(dir, "BepInEx")}
			if c.renderer {
				preloader := filepath.Join(dir, "Renderer", "BepInEx", "core", "BepInEx.Preloader.dll")
				touch(t, preloader)
				if c.doorstopWine {
					preloader = wineoverride.ZPath(preloader)
				}
				want = append(want, "--doorstop-enabled", "true", "--doorstop-target-assembly", preloader)
			}
			plan := launchplan.New(launchplan.ModeProfile)
			if err := l.Contribute(context.Background(), plan, view(dir, c.runtime)); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(plan.Args, want) {
				t.Errorf("args = %q, want %q", plan.Args, want)
			}
			if len(plan.Files) != len(beside) {
				t.Errorf("placed %d files, want %d", len(plan.Files), len(beside))
			}
			if len(plan.RuntimeReqs) != 0 {
				t.Errorf("runtime requirements = %+v: hostfxr needs no override", plan.RuntimeReqs)
			}
		})
	}
}

func TestAProfileWithoutTheShim(t *testing.T) {
	l, dir := installed(t, realPack...)
	plan := launchplan.New(launchplan.ModeProfile)
	if err := l.Contribute(context.Background(), plan, view(dir, "proton")); err != nil {
		t.Fatal(err)
	}
	if want := []string{"--hookfxr-disable", "--doorstop-enabled", "false", "--bepinex-target", filepath.Join(dir, "BepInEx")}; !slices.Equal(plan.Args, want) {
		t.Errorf("no mods: args = %q, want %q", plan.Args, want)
	}
	if len(plan.Files) != 0 {
		t.Errorf("no mods: placed %+v", plan.Files)
	}
	if !l.Owns(loader.Process{Args: plan.Args}, view(dir, "proton")) {
		t.Error("no mods: the profile does not own its process")
	}

	touch(t, filepath.Join(dir, "BepInEx", "plugins", "Ns-Mod", "Mod.dll"))
	if err := l.Contribute(context.Background(), launchplan.New(launchplan.ModeProfile), view(dir, "proton")); err == nil || !strings.Contains(err.Error(), "BepInExResoniteShim") {
		t.Errorf("mods without the shim: err = %v", err)
	}
}

func TestVanilla(t *testing.T) {
	plan := launchplan.New(launchplan.ModeVanilla)
	if err := New().Vanilla(context.Background(), plan, loader.Target{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"--hookfxr-disable", "--doorstop-enabled", "false"}; !slices.Equal(plan.Args, want) {
		t.Errorf("args = %q, want %q", plan.Args, want)
	}
}

func TestOwnsOnlyItsOwnProfile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "default")
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"--hookfxr-enable", "--bepinex-target", filepath.Join(dir, "BepInEx")}, true},
		{[]string{"--bepinex-target", filepath.Join(dir+"-2", "BepInEx")}, false},
		{[]string{"--hookfxr-disable", "--doorstop-enabled", "false"}, false},
		{[]string{"--bepinex-target"}, false},
	}
	for _, c := range cases {
		if got := New().Owns(loader.Process{Args: c.args}, view(dir, "proton")); got != c.want {
			t.Errorf("Owns(%q) = %v", c.args, got)
		}
	}
}

// coreCLRLog is LogOutput.log as BepInEx.Core 6 words it (format strings read from the pack's BepInEx.Core.dll), with
// a .NET stack in CoreCLR's format.
const coreCLRLog = `[Message:   BepInEx] BepInEx 6.0.0-be - Renderite.Host
[Info   :   BepInEx] 2 plugins to load
[Info   :   BepInEx] Loading [Good Mod 1.0.0]
[Error  :   BepInEx] Could not load [Needy 1.0.0] because it has missing dependencies: net.gone.mod. Install the listed plugin(s) and restart the game.
[Error  :   BepInEx] Error loading [Broken Mod 1.0.0]: System.TypeLoadException: Could not load type
[Message:   BepInEx] Chainloader startup complete
System.NullReferenceException: Object reference not set to an instance of an object.
   at CoolMod.Patches.Tick(World world) in D:\src\CoolMod\Patches.cs:line 12
   at FrooxEngine.World.Update()
`

func TestACoreCLRLogReads(t *testing.T) {
	l := New()
	if !l.Ready("[Message:   BepInEx] Chainloader startup complete") {
		t.Error("Ready missed BepInEx's last startup line")
	}
	if _, n := l.Load("[Info   :   BepInEx] 2 plugins to load"); n != 2 {
		t.Errorf("total = %d", n)
	}
	if m, _ := l.Load("[Info   :   BepInEx] Loading [Good Mod 1.0.0]"); m != "Good Mod 1.0.0" {
		t.Errorf("loading = %q", m)
	}
	found := map[string]string{}
	for _, f := range l.Analyzers()[0].Analyze(loader.Logs{Loader: coreCLRLog}) {
		found[f.Plugin] = f.Kind
	}
	if found["Needy 1.0.0"] != bepinex5.KindMissingDependency || found["Broken Mod 1.0.0"] != bepinex5.KindLoadException {
		t.Errorf("findings = %v", found)
	}
	blames := l.StackBlame(coreCLRLog)
	if len(blames) != 1 || !strings.HasPrefix(blames[0].Namespace, "CoolMod") {
		t.Errorf("blames = %+v", blames)
	}
	if n := bepinex5.Unclassified(coreCLRLog); n != 0 {
		t.Errorf("%d lines unclassified", n)
	}
}

func TestLogAndProfileFilesAreBepInExs(t *testing.T) {
	l := New()
	p, err := l.Path(view("/p", "native"))
	if err != nil || p != filepath.Join("/p", "BepInEx", "LogOutput.log") {
		t.Errorf("log = %q, %v", p, err)
	}
	files := l.ProfileFiles()
	for _, want := range append(slices.Clone(beside), "BepInEx", "Renderer") {
		if !slices.Contains(files, want) {
			t.Errorf("ProfileFiles lacks %s", want)
		}
	}
	if d := l.OverrideDLL("resonite"); d != "" {
		t.Errorf("OverrideDLL = %q", d)
	}
	if !slices.Contains(l.ConfigDirs(), "Renderer/BepInEx/config") {
		t.Errorf("ConfigDirs = %v", l.ConfigDirs())
	}
}
