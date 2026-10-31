package bepinex5

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
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
	if err := safefs.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
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
	testfs.WriteAt(t, cfg, []byte(orig))
	if err := l.SetLaunchSetting(dir, "console", "false"); err != nil {
		t.Fatal(err)
	}
	if err := l.SetLaunchSetting(dir, "logLevel", "debug"); err != nil {
		t.Fatal(err)
	}
	b, _ := safefs.ReadFile(cfg)
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
	testfs.WriteAt(t, cfg, []byte("[Logging.Console]\n\nEnabled = true\n"))
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
