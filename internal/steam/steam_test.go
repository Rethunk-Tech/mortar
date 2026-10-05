package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"
)

// fixture copies testdata/lib into a temp Steam root, pointing the library list at it.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("testdata/lib")); err != nil {
		t.Fatal(err)
	}
	vdfPath := filepath.Join(root, "steamapps", "libraryfolders.vdf")
	b, err := fsx.ReadFile(vdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(vdfPath, []byte(strings.ReplaceAll(string(b), "@ROOT@", root)), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestLocate(t *testing.T) {
	mk := func(rel string) string {
		home := t.TempDir()
		if err := os.MkdirAll(filepath.Join(home, rel), 0o700); err != nil {
			t.Fatal(err)
		}
		return home
	}
	for _, tc := range []struct {
		name string
		home string
		want Status
		kind Kind
	}{
		{"share", mk(".local/share/Steam/steamapps"), Found, KindNative},
		{"dot steam", mk(".steam/steam/steamapps"), Found, KindNative},
		{"flatpak steam", mk(".var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps"), Found, KindFlatpak},
		{"flatpak only", mk(".var/app/com.valvesoftware.Steam/.local/share/Steam"), FlatpakOnly, KindNative},
		{"missing", t.TempDir(), NotFound, KindNative},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, got := Locate(tc.home)
			if got != tc.want {
				t.Fatalf("status = %s, want %s", got, tc.want)
			}
			if got != Found && s.Root != "" {
				t.Fatalf("root = %q for %s", s.Root, got)
			}
			if got == Found && s.Kind != tc.kind {
				t.Fatalf("kind = %q, want %q", s.Kind, tc.kind)
			}
			if got == Found && s.Root == "" {
				t.Fatal("found with empty root")
			}
		})
	}
	both := mk(".local/share/Steam/steamapps")
	if err := os.MkdirAll(filepath.Join(both, ".var/app/com.valvesoftware.Steam/.local/share/Steam/steamapps"), 0o700); err != nil {
		t.Fatal(err)
	}
	s, st := Locate(both)
	if st != Found || s.Kind != KindNative {
		t.Fatalf("native wins: %+v %s", s, st)
	}
	all := LocateAll(both)
	if len(all) != 2 || all[0].Kind != KindNative || all[1].Kind != KindFlatpak {
		t.Fatalf("all = %#v", all)
	}
}

func TestFixture(t *testing.T) {
	root := fixture(t)
	s := Steam{Root: root}

	libs, err := s.Libraries()
	if err != nil || len(libs) != 2 {
		t.Fatalf("libraries = %v, %v", libs, err)
	}

	dir, err := s.InstallDir("413150")
	if err != nil || dir != filepath.Join(root, "steamapps", "common", "Stardew Valley") {
		t.Fatalf("install dir = %q, %v", dir, err)
	}
	if dir, err := s.InstallDir("1966720"); err != nil || dir != "" {
		t.Fatalf("uninstalled app: %q, %v", dir, err)
	}

	acct, err := s.CurrentAccount()
	if err != nil || acct.AccountName != "newuser" || acct.ID != "76561190000000002" {
		t.Fatalf("account = %+v, %v", acct, err)
	}

	if got := s.HeroArt("413150"); got != filepath.Join(root, "appcache", "librarycache", "413150", "library_hero.jpg") {
		t.Fatalf("hero = %q", got)
	}
	if got := s.HeroArt("1966720"); got != filepath.Join(root, "appcache", "librarycache", "1966720_library_hero.jpg") {
		t.Fatalf("fallback hero = %q", got)
	}
	if got := s.HeroArt("1"); got != "" {
		t.Fatalf("missing hero = %q", got)
	}
}

func TestLaunchOptions(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("config/loginusers.vdf", `"users" { "76561198000000001" { "AccountName" "a" "MostRecent" "0" } "76561198000000002" { "AccountName" "b" "MostRecent" "1" } }`)
	s := Steam{Root: root}
	if _, err := s.LaunchOptions("413150"); err == nil {
		t.Fatal("missing localconfig.vdf should fail")
	}
	// 76561198000000002 - 76561197960265728 = 39734274
	cfg := "userdata/39734274/config/localconfig.vdf"
	write(cfg, `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" {
		"413150" { "LaunchOptions" "\"C:\\Games\\Stardew Valley\\StardewModdingAPI.exe\" %command%" }
		"1966720" { "LastPlayed" "1" } } } } } }`)
	got, err := s.LaunchOptions("413150")
	if err != nil || !strings.Contains(got, "StardewModdingAPI.exe") || !strings.Contains(got, "%command%") {
		t.Fatalf("launch options = %q, %v", got, err)
	}
	if got, err := s.LaunchOptions("1966720"); err != nil || got != "" {
		t.Fatalf("app without options = %q, %v", got, err)
	}
	if got, err := s.LaunchOptions("1"); err != nil || got != "" {
		t.Fatalf("unknown app = %q, %v", got, err)
	}
}

func TestRunFlatpakGoesThroughHostInFlatpak(t *testing.T) {
	oldOut, oldEnv := sandbox.Output, sandbox.Getenv
	t.Cleanup(func() { sandbox.Output, sandbox.Getenv = oldOut, oldEnv })
	sandbox.Getenv = func(string) string { return "x" }
	var got []string
	sandbox.Output = func(args ...string) ([]byte, error) { got = args; return nil, nil }
	if _, err := ShowOverride(); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, " ") != "--host flatpak override --user --show "+FlatpakID {
		t.Fatalf("argv %v", got)
	}
}

func TestBuildIDReadsTheLibrariesAppManifest(t *testing.T) {
	lib := t.TempDir()
	if err := os.MkdirAll(filepath.Join(lib, "steamapps", "common", "Game"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, "steamapps", "appmanifest_7.acf"), []byte("\"AppState\"\n{\n\t\"buildid\"\t\t\"123\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := BuildID(filepath.Join(lib, "steamapps", "common", "Game"), "7"); got != "123" {
		t.Fatalf("BuildID = %q", got)
	}
}
