package bepinex5

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

var _ interface{ OverrideDLL(string) string } = Loader{}

func buildPack(t *testing.T, doorstop string) string { return buildPackIn(t, "BepInExPack", doorstop) }

// buildPackIn builds a pack whose files sit under root, as a community build names it (BepInExPack_Valheim).
func buildPackIn(t *testing.T, root, doorstop string) string {
	t.Helper()
	return buildPackWithProxy(t, root, doorstop, "winhttp.dll")
}

// buildPackWithProxy builds a pack whose Doorstop proxy has the given name.
func buildPackWithProxy(t *testing.T, root, doorstop, proxy string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pack.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"manifest.json":                      `{"name":"BepInExPack","version_number":"5.4.2305"}`,
		"README.md":                          "pack",
		proxy:                                "proxy",
		"doorstop_config.ini":                "[UnityDoorstop]",
		".doorstop_version":                  doorstop,
		"doorstop_libs/x64/libdoorstop.so":   "lib",
		"BepInEx/core/BepInEx.Preloader.dll": "pre",
	} {
		if name == ".doorstop_version" && doorstop == "" {
			continue
		}
		if root != "" && name != "manifest.json" && name != "README.md" {
			name = root + "/" + name
		}
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInstallPackAndDoorstopFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	// A reinstall replaces in place; "" is a pack with BepInEx/ at the zip's top (bbepis-BepInEx_Rogue_Tower).
	for _, packDir := range []string{"BepInExPack", "BepInExPack", "BepInExPack_Valheim", ""} {
		got, err := InstallPack(buildPackIn(t, packDir, "4.3.0.0\n"), root)
		if err != nil || got.Version != "5.4.2305" || got.Doorstop != 4 {
			t.Fatalf("%s: got %+v, %v", packDir, got, err)
		}
	}
	if !fsx.IsDir(filepath.Join(root, "BepInEx", "core")) {
		t.Fatal("BepInEx/core missing")
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err == nil {
		t.Fatal("pack metadata leaked into the profile")
	}
	want := []string{"doorstop_config.ini", "winhttp.dll"}
	if got := DoorstopFiles(root); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("doorstop files %v", got)
	}
	if got, _ := InstallPack(buildPack(t, "garbage"), filepath.Join(t.TempDir(), "p")); got.Doorstop != 3 {
		t.Fatalf("default doorstop %d", got.Doorstop)
	}
}

func TestAnOlderPackOverANewerOneKeepsNoneOfItsFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	if _, err := InstallPack(buildPack(t, "4.3.0.0\n"), root); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"BepInEx/core/Stale.dll":         "newer pack only",
		"BepInEx/core/Ns-Mod/Mod.dll":    "a package's core file",
		"BepInEx/config/BepInEx.cfg":     "[Logging.Console]\nEnabled = true\n",
		"BepInEx/plugins/Ns-Mod/Mod.dll": "plugin",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := fsx.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := InstallPack(buildPack(t, ""), root)
	if err != nil || got.Doorstop != 3 {
		t.Fatalf("an older pack without .doorstop_version is Doorstop 3: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, "BepInEx", "core", "Stale.dll")); err == nil {
		t.Fatal("the newer pack's core file was left behind")
	}
	for _, kept := range []string{"BepInEx/core/Ns-Mod/Mod.dll", "BepInEx/plugins/Ns-Mod/Mod.dll", "BepInEx/core/BepInEx.Preloader.dll"} {
		if _, err := os.Stat(filepath.Join(root, kept)); err != nil {
			t.Fatalf("%s: %v", kept, err)
		}
	}
	if b, _ := fsx.ReadFile(filepath.Join(root, "BepInEx", "config", "BepInEx.cfg")); !strings.Contains(string(b), "Enabled = true") {
		t.Fatalf("the player's BepInEx.cfg was replaced: %q", b)
	}
}

// An IL2CPP pack (BepInEx 6, BepInExPack_IL2CPP) starts through its own entry assembly on the .NET runtime it ships,
// which Doorstop finds only through absolute paths, since it reads relative ones from the game folder.
func TestAnIL2CPPPackStartsThroughItsRuntime(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pack.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"manifest.json":                                     `{"name":"BepInExPack_IL2CPP","version_number":"6.0.788"}`,
		"BepInExPack/winhttp.dll":                           "proxy",
		"BepInExPack/.doorstop_version":                     "4.5.0",
		"BepInExPack/doorstop_config.ini":                   "[General]\r\ntarget_assembly = BepInEx\\core\\BepInEx.Unity.IL2CPP.dll\r\n[Il2Cpp]\r\ncoreclr_path = dotnet\\coreclr.dll\r\ncorlib_dir = dotnet\r\n",
		"BepInExPack/BepInEx/core/BepInEx.Unity.IL2CPP.dll": "entry",
		"BepInExPack/dotnet/coreclr.dll":                    "clr",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	tgt := loader.Target{ProfileDir: filepath.Join(t.TempDir(), "profile")}
	if ver, err := (Loader{}).Install(ctx, tgt, loader.Package{Archive: p}, nil); err != nil || ver != "6.0.788" {
		t.Fatalf("install %q %v", ver, err)
	}
	if st, _ := (Loader{}).Status(tgt); !st.Installed {
		t.Fatal("an IL2CPP pack reads as not installed")
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(ctx, plan, loader.ProfileView{Dir: tgt.ProfileDir, Runtime: "proton"}); err != nil {
		t.Fatal(err)
	}
	z := `Z:` + strings.ReplaceAll(filepath.ToSlash(tgt.ProfileDir), "/", `\`)
	want := "--doorstop-enabled true --doorstop-target-assembly " + z + `\BepInEx\core\BepInEx.Unity.IL2CPP.dll` +
		" --doorstop-clr-corlib-dir " + z + `\dotnet --doorstop-clr-runtime-coreclr-path ` + z + `\dotnet\coreclr.dll`
	if got := strings.Join(plan.Args, " "); got != want {
		t.Fatalf("args\n got %q\nwant %q", got, want)
	}
	cfg, err := os.ReadFile(plan.Files[0].Src)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"corlib_dir=" + z + `\dotnet` + "\r", "coreclr_path=" + z + `\dotnet\coreclr.dll` + "\r"} {
		if !strings.Contains(string(cfg), line) {
			t.Fatalf("config lacks %q:\n%s", line, cfg)
		}
	}
	if _, err := InstallPack(buildPack(t, "4.3.0.0"), tgt.ProfileDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(tgt.ProfileDir, "dotnet")); err == nil || hasCLR(tgt.ProfileDir) {
		t.Fatal("a BepInEx 5 pack over an IL2CPP one kept its runtime")
	}
}

// BepInEx 6 Mono (Slip & Skid, a zip written with backslash separators) and the legacy IL2CPP build on a shipped Mono
// runtime (Crab Game) start through their own entry assembly, and their config's relative runtime folders point into
// the profile, since Doorstop reads them from the game folder.
func TestPacksWithTheirOwnRuntimeFolders(t *testing.T) {
	cases := []struct {
		name, sep, entry, config string
		files                    []string
		want                     []string
	}{
		{
			"mono", `\`, `BepInEx\core\BepInEx.Unity.Mono.Preloader.dll`,
			"[General]\ntarget_assembly = BepInEx\\core\\BepInEx.Unity.Mono.Preloader.dll\n[UnityMono]\ndll_search_path_override = unstripped_corlib\n",
			[]string{`unstripped_corlib\mscorlib.dll`},
			[]string{"dll_search_path_override=%s/unstripped_corlib", "target_assembly=%s/BepInEx/core/BepInEx.Unity.Mono.Preloader.dll"},
		},
		{
			"il2cpp-on-mono", "/", "BepInEx/core/BepInEx.IL2CPP.dll",
			"[UnityDoorstop]\ntargetAssembly=BepInEx\\core\\BepInEx.IL2CPP.dll\n[MonoBackend]\nruntimeLib=mono\\EmbedRuntime\\mono-2.0-sgen.dll\nconfigDir=mono\\etc\ncorlibDir=mono\\Managed\n",
			[]string{"mono/EmbedRuntime/mono-2.0-sgen.dll", "mono/etc/machine.config", "mono/Managed/mscorlib.dll"},
			[]string{"runtimeLib=%s/mono/EmbedRuntime/mono-2.0-sgen.dll", "configDir=%s/mono/etc", "corlibDir=%s/mono/Managed", "targetAssembly=%s/BepInEx/core/BepInEx.IL2CPP.dll"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			add := func(name, body string) {
				w, _ := zw.Create("BepInExPack" + c.sep + name)
				_, _ = w.Write([]byte(body))
			}
			if c.sep != "/" {
				add(`BepInEx\`, "")
			}
			add("winhttp.dll", "proxy")
			add("doorstop_config.ini", c.config)
			add(c.entry, "entry")
			for _, f := range c.files {
				add(f, "x")
			}
			m, _ := zw.Create("manifest.json")
			_, _ = m.Write([]byte(`{"name":"BepInExPack","version_number":"6.0.1"}`))
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			zipPath := filepath.Join(t.TempDir(), "pack.zip")
			if err := fsx.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			tgt := loader.Target{ProfileDir: filepath.Join(t.TempDir(), "profile")}
			if _, err := (Loader{}).Install(t.Context(), tgt, loader.Package{Archive: zipPath}, nil); err != nil {
				t.Fatal(err)
			}
			if st, _ := (Loader{}).Status(tgt); !st.Installed {
				t.Fatal("the pack reads as not installed")
			}
			cfgPath, err := absoluteDoorstopConfig(tgt.ProfileDir, 4, false)
			if err != nil {
				t.Fatal(err)
			}
			cfg, _ := fsx.ReadFile(cfgPath)
			abs := filepath.ToSlash(tgt.ProfileDir)
			for _, line := range c.want {
				if !strings.Contains(strings.ReplaceAll(string(cfg), `\`, "/"), fmt.Sprintf(line, abs)) {
					t.Errorf("config lacks %q:\n%s", fmt.Sprintf(line, abs), cfg)
				}
			}
		})
	}
}

func TestLaunchArgs(t *testing.T) {
	root := "/home/u/mortar/profile one"
	pre := "/home/u/mortar/profile one/BepInEx/core/BepInEx.Preloader.dll"
	cases := []struct {
		major  int
		proton bool
		want   string
	}{
		{3, false, "--doorstop-enable true --doorstop-target " + pre},
		{4, false, "--doorstop-enabled true --doorstop-target-assembly " + pre},
		{4, true, `--doorstop-enabled true --doorstop-target-assembly Z:\home\u\mortar\profile one\BepInEx\core\BepInEx.Preloader.dll`},
	}
	for _, c := range cases {
		if got := strings.Join(LaunchArgs(root, c.major, c.proton), " "); got != c.want {
			t.Errorf("LaunchArgs(%d,%v) = %q, want %q", c.major, c.proton, got, c.want)
		}
	}
	if strings.Join(VanillaArgs(), " ") != "--doorstop-enable false --doorstop-enabled false" {
		t.Fatal("vanilla args")
	}
}

func TestRoute(t *testing.T) {
	const pkg = "Ns-Mod"
	cases := map[string]string{
		"Mod.dll":                         "BepInEx/plugins/Ns-Mod/Mod.dll",
		"Assets/x.bundle":                 "BepInEx/plugins/Ns-Mod/x.bundle",
		"FSharp.Core/FSharp.Core.dll":     "BepInEx/plugins/Ns-Mod/FSharp.Core.dll",
		"Ns-Mod/BepInEx/plugins/a/b.dll":  "BepInEx/plugins/Ns-Mod/a/b.dll",
		"Extra/Config/Ns.Mod.cfg":         "BepInEx/config/Ns.Mod.cfg",
		"BEPINEX/Plugins/Deep/x.dll":      "BepInEx/plugins/Ns-Mod/Deep/x.dll",
		"lib/Fix.mm.dll":                  "BepInEx/monomod/Ns-Mod/Fix.mm.dll",
		"docs/README.md":                  "BepInEx/plugins/Ns-Mod/README.md",
		"plugins/Mod.dll":                 "BepInEx/plugins/Ns-Mod/Mod.dll",
		"BepInEx/plugins/sub/Mod.dll":     "BepInEx/plugins/Ns-Mod/sub/Mod.dll",
		"config/ns.mod.cfg":               "BepInEx/config/ns.mod.cfg",
		"BepInEx/config/ns.mod.cfg":       "BepInEx/config/ns.mod.cfg",
		"patchers/Patch.dll":              "BepInEx/patchers/Ns-Mod/Patch.dll",
		"BepInEx/monomod/Game.Fix.mm.dll": "BepInEx/monomod/Ns-Mod/Game.Fix.mm.dll",
		"Game.Fix.mm.dll":                 "BepInEx/monomod/Ns-Mod/Game.Fix.mm.dll",
		"core/Lib.dll":                    "BepInEx/core/Ns-Mod/Lib.dll",
		"manifest.json":                   "BepInEx/plugins/Ns-Mod/manifest.json",
		"README.md":                       "BepInEx/plugins/Ns-Mod/README.md",
		"Mod.pdb":                         "BepInEx/plugins/Ns-Mod/Mod.pdb",
		"plugins/Sub/Mod.dll.mdb":         "BepInEx/plugins/Ns-Mod/Sub/Mod.dll.mdb",
		"../evil.dll":                     "",
		"config/../../evil.cfg":           "",
		`plugins\..\..\evil.dll`:          "",
		"C:/Windows/evil.dll":             "",
		"plugins/Mod.dll:stream":          "",
	}
	for in, want := range cases {
		if got := Route(in, pkg, nil); got != want {
			t.Errorf("Route(%q) = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "..", "../x", `a\b`, "C:x"} {
		if got := Route("Mod.dll", bad, nil); got != "" {
			t.Errorf("Route with package %q = %q, want none", bad, got)
		}
	}
}

func TestRouteCatalogRoutes(t *testing.T) {
	routes := []components.LoaderRoute{
		{Dir: "BepInEx/UMM Mods", Track: components.TrackSubdir},
		{Dir: "BepInEx/Sideloader", Extensions: []string{".hotmod", ".h3mod"}},
		{Dir: "BepInEx/Skins", Extensions: []string{".nasbskin"}, Track: components.TrackSubdir},
		{Dir: "QMods"},
	}
	cases := map[string]string{
		"UMM Mods/Cool/Cool.dll":       "BepInEx/UMM Mods/Ns-Mod/Cool/Cool.dll",
		"x/umm mods/Cool.dll":          "BepInEx/UMM Mods/Ns-Mod/Cool.dll",
		"BepInEx/Sideloader/a/b.h3mod": "BepInEx/Sideloader/a/b.h3mod",
		"deep/folder/Thing.HotMod":     "BepInEx/Sideloader/Thing.HotMod",
		"pack/Fancy.nasbskin":          "BepInEx/Skins/Ns-Mod/Fancy.nasbskin",
		"QMods/Mod/mod.json":           "QMods/Mod/mod.json",
		"plugins/UMM Mods/x.dll":       "BepInEx/plugins/Ns-Mod/UMM Mods/x.dll",
		"Mod.dll":                      "BepInEx/plugins/Ns-Mod/Mod.dll",
		"QMods/../../evil.dll":         "",
		`QMods\..\..\evil.dll`:         "",
		"C:/QMods/evil.dll":            "",
		"QMods/Mod:stream":             "",
	}
	for in, want := range cases {
		if got := Route(in, "Ns-Mod", routes); got != want {
			t.Errorf("Route(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLaunchSettingsEditOnlyTheirLines(t *testing.T) {
	dir := t.TempDir()
	l := Loader{}
	got, err := l.LaunchSettings(dir)
	if err != nil || got[0].Value != "false" || got[1].Value != "default" {
		t.Fatalf("defaults = %+v, %v", got, err)
	}
	cfg := configFile(dir)
	orig := "## kept\r\n[Logging.Console]\r\n\r\nEnabled = true\r\nLogLevels = Fatal, Error, Warning, Message, Info\r\n\r\n[Other]\r\nX = 1\r\n"
	if err := os.MkdirAll(filepath.Dir(cfg), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := l.SetLaunchSetting(dir, "console", "false"); err != nil {
		t.Fatal(err)
	}
	if err := l.SetLaunchSetting(dir, "logLevel", "debug"); err != nil {
		t.Fatal(err)
	}
	b, _ := fsx.ReadFile(cfg)
	want := "## kept\r\n[Logging.Console]\r\n\r\nEnabled = false\r\nLogLevels = Fatal, Error, Warning, Message, Info, Debug\r\n\r\n[Other]\r\nX = 1\r\n\r\n[Logging.Disk]\r\n\r\nLogLevels = Fatal, Error, Warning, Message, Info, Debug\r\n"
	if string(b) != want {
		t.Fatalf("cfg = %q\nwant %q", b, want)
	}
	got, _ = l.LaunchSettings(dir)
	if got[0].Value != "false" || got[1].Value != "debug" {
		t.Fatalf("read back = %+v", got)
	}
	if l.SetLaunchSetting(dir, "logLevel", "loud") == nil || l.SetLaunchSetting(dir, "nope", "x") == nil {
		t.Fatal("unknown values must be refused")
	}
}

// BepInEx writes Enabled = true into a new BepInEx.cfg, so every real launch, never a preview, puts back the user's
// choice, off by default, and a choice to show the window survives a reinstall of the pack.
func TestLaunchHidesTheConsoleWindowUnlessChosen(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	cfg := configFile(dir)
	if err := os.MkdirAll(filepath.Dir(cfg), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[Logging.Console]\n\nEnabled = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	l := Loader{}
	before, _ := readConfig(dir)
	if err := l.Contribute(ctx, launchplan.New(launchplan.ModeProfile), loader.ProfileView{Dir: dir}); err != nil {
		t.Fatal(err)
	}
	if after, _ := readConfig(dir); after != before {
		t.Fatalf("building the plan, as a preview does, rewrote BepInEx.cfg: %q", after)
	}
	enabled := func() string {
		if err := l.Prelaunch(loader.ProfileView{Dir: dir}); err != nil {
			t.Fatal(err)
		}
		text, _ := readConfig(dir)
		return cfgGet(text, "Logging.Console", "Enabled")
	}
	if got := enabled(); got != "false" {
		t.Fatalf("default launch: Enabled = %q", got)
	}
	if err := l.SetLaunchSetting(dir, "console", "true"); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Install(ctx, loader.Target{ProfileDir: dir}, loader.Package{Archive: buildPack(t, "4.3.0.0")}, nil); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[Logging.Console]\n\nEnabled = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := enabled(); got != "true" {
		t.Fatalf("chosen console: Enabled = %q", got)
	}
}

func TestAPackWhoseProxyIsVersionDllLoadsThatOne(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	if _, err := (Loader{}).Install(t.Context(), loader.Target{ProfileDir: profile}, loader.Package{Archive: buildPackWithProxy(t, "BepInExPack", "4.5.0", "version.dll")}, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(DoorstopFiles(profile), " "); got != "doorstop_config.ini version.dll" {
		t.Fatalf("placed %q", got)
	}
	plan := launchplan.New(launchplan.ModeProfile)
	if err := (Loader{}).Contribute(t.Context(), plan, loader.ProfileView{Dir: profile, Runtime: "proton"}); err != nil {
		t.Fatal(err)
	}
	if len(plan.RuntimeReqs) != 1 || plan.RuntimeReqs[0].Key != "version" {
		t.Fatalf("overrides %+v: Wine must load the pack's own proxy natively", plan.RuntimeReqs)
	}
	// An older pack with the usual proxy replaces the renamed one rather than leaving both.
	if _, err := (Loader{}).Install(t.Context(), loader.Target{ProfileDir: profile}, loader.Package{Archive: buildPack(t, "4.3.0.0")}, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(DoorstopFiles(profile), " "); got != "doorstop_config.ini winhttp.dll" || fsx.IsFile(filepath.Join(profile, "version.dll")) {
		t.Fatalf("placed %q after the swap", got)
	}
}

func TestTheCatalogNamesTheProxyBeforeAProfileHoldsAPack(t *testing.T) {
	if got, other := (Loader{}).OverrideDLL("in-silence"), (Loader{}).OverrideDLL("lethal-company"); got != "version" || other != "winhttp" {
		t.Fatalf("OverrideDLL = %q for In Silence, %q for Lethal Company", got, other)
	}
	plan := launchplan.New(launchplan.ModeProfile)
	view := loader.ProfileView{Game: "in-silence", Dir: t.TempDir(), Runtime: "proton"}
	if err := (Loader{}).Contribute(t.Context(), plan, view); err != nil {
		t.Fatal(err)
	}
	if len(plan.RuntimeReqs) != 1 || plan.RuntimeReqs[0].Key != "version" {
		t.Fatalf("overrides %+v", plan.RuntimeReqs)
	}
}
