package bepinex5

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

var _ interface{ NeedsWinHTTPOverride() bool } = Loader{}

func buildPack(t *testing.T, doorstop string) string { return buildPackIn(t, "BepInExPack", doorstop) }

// buildPackIn builds a pack whose files sit under root, as a community build names it (BepInExPack_Valheim).
func buildPackIn(t *testing.T, root, doorstop string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pack.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"manifest.json":                      `{"name":"BepInExPack","version_number":"5.4.2305"}`,
		"README.md":                          "pack",
		"winhttp.dll":                        "proxy",
		"doorstop_config.ini":                "[UnityDoorstop]",
		".doorstop_version":                  doorstop,
		"doorstop_libs/x64/libdoorstop.so":   "lib",
		"BepInEx/core/BepInEx.Preloader.dll": "pre",
	} {
		if name == ".doorstop_version" && doorstop == "" {
			continue
		}
		if name != "manifest.json" && name != "README.md" {
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
	for _, packDir := range []string{"BepInExPack", "BepInExPack", "BepInExPack_Valheim"} { // a reinstall replaces in place
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

func TestEnsureWinHTTPOverride(t *testing.T) {
	fixture, err := fsx.ReadFile("testdata/user.reg")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"section present":   string(fixture),
		"section absent":    strings.Replace(string(fixture), "[Software\\\\Wine\\\\DllOverrides]", "[Software\\\\Wine\\\\Other]", 1),
		"stale value":       strings.Replace(string(fixture), `"dxgi"="native,builtin"`, `"winhttp"="builtin"`, 1),
		"crlf line endings": strings.ReplaceAll(string(fixture), "\n", "\r\n"),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			reg := filepath.Join(t.TempDir(), "user.reg")
			if err := fsx.WriteFile(reg, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := EnsureWinHTTPOverride(reg); err != nil {
				t.Fatal(err)
			}
			once, _ := fsx.ReadFile(reg)
			if err := EnsureWinHTTPOverride(reg); err != nil {
				t.Fatal(err)
			}
			twice, _ := fsx.ReadFile(reg)
			if string(once) != string(twice) || strings.Count(string(once), `"winhttp"="native,builtin"`) != 1 {
				t.Fatalf("not idempotent or duplicated:\n%s", once)
			}
			if backup, _ := fsx.ReadFile(reg + ".mortar-backup"); string(backup) != content {
				t.Fatal("backup is not the original")
			}
			// Every other line survives, in order, with the file's own line ending.
			eol := "\n"
			if strings.Contains(content, "\r\n") {
				eol = "\r\n"
			}
			if strings.Contains(strings.ReplaceAll(string(once), eol, ""), "\n") {
				t.Fatal("line endings mixed")
			}
			rest := strings.Split(strings.Replace(string(once), `"winhttp"="native,builtin"`+eol, "", 1), eol)
			for l := range strings.SplitSeq(content, eol) {
				if l != "" && !strings.Contains(strings.Join(rest, eol), l) && !strings.HasPrefix(l, `"winhttp"=`) && !strings.HasPrefix(l, `[Software\\Wine\\Other]`) {
					t.Fatalf("line lost: %q", l)
				}
			}
		})
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
		"manifest.json":                   "",
		"README.md":                       "",
		"../evil.dll":                     "",
		"config/../../evil.cfg":           "",
		`plugins\..\..\evil.dll`:          "",
		"C:/Windows/evil.dll":             "",
		"plugins/Mod.dll:stream":          "",
	}
	for in, want := range cases {
		if got := Route(in, pkg); got != want {
			t.Errorf("Route(%q) = %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "..", "../x", `a\b`, "C:x"} {
		if got := Route("Mod.dll", bad); got != "" {
			t.Errorf("Route with package %q = %q, want none", bad, got)
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
