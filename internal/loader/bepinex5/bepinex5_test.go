package bepinex5

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

var _ interface{ NeedsWinHTTPOverride() bool } = Loader{}

func buildPack(t *testing.T, doorstop string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "pack.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"manifest.json":                                  `{"name":"BepInExPack","version_number":"5.4.2305"}`,
		"README.md":                                      "pack",
		"BepInExPack/winhttp.dll":                        "proxy",
		"BepInExPack/doorstop_config.ini":                "[UnityDoorstop]",
		"BepInExPack/.doorstop_version":                  doorstop,
		"BepInExPack/doorstop_libs/x64/libdoorstop.so":   "lib",
		"BepInExPack/BepInEx/core/BepInEx.Preloader.dll": "pre",
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
	return p
}

func TestInstallPackAndDoorstopFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	for range 2 { // a second install replaces in place
		got, err := InstallPack(buildPack(t, "4.3.0.0\n"), root)
		if err != nil || got.Version != "5.4.2305" || got.Doorstop != 4 {
			t.Fatalf("got %+v, %v", got, err)
		}
	}
	if !fsx.IsDir(filepath.Join(root, "BepInEx", "core")) {
		t.Fatal("BepInEx/core missing")
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err == nil {
		t.Fatal("pack metadata leaked into the profile")
	}
	want := []string{".doorstop_version", "doorstop_config.ini", "doorstop_libs/x64/libdoorstop.so", "winhttp.dll"}
	if got := DoorstopFiles(root); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("doorstop files %v", got)
	}
	if got, _ := InstallPack(buildPack(t, "garbage"), filepath.Join(t.TempDir(), "p")); got.Doorstop != 3 {
		t.Fatalf("default doorstop %d", got.Doorstop)
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
	if strings.Join(VanillaArgs(3), " ") != "--doorstop-enable false" || strings.Join(VanillaArgs(4), " ") != "--doorstop-enabled false" {
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
		"Assets/x.bundle":                 "BepInEx/plugins/Ns-Mod/Assets/x.bundle",
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
