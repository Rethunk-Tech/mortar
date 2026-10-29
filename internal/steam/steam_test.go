package steam

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/go-safefs"

	"github.com/andygrunwald/vdf"
)

// fixture copies testdata/lib into a temp Steam root, pointing the library list at it.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("testdata/lib")); err != nil {
		t.Fatal(err)
	}
	vdfPath := filepath.Join(root, "steamapps", "libraryfolders.vdf")
	b, err := safefs.ReadFile(vdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(vdfPath, []byte(strings.ReplaceAll(string(b), "@ROOT@", strings.ReplaceAll(root, `\`, `\\`))), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

// nativeRoots names home's native Steam folder as a player-added root on Windows, where Steam is found through the
// registry and not under home.
func nativeRoots(home string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	return []string{filepath.Join(home, ".local", "share", "Steam")}
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
			if runtime.GOOS == "windows" && tc.name == "dot steam" {
				t.Skip("the ~/.steam links are Linux's")
			}
			var extra []string
			if tc.want == Found && tc.kind == KindNative {
				extra = nativeRoots(tc.home)
			}
			s, got := Locate(tc.home, extra...)
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
	s, st := Locate(both, nativeRoots(both)...)
	if st != Found || s.Kind != KindNative {
		t.Fatalf("native wins: %+v %s", s, st)
	}
	all := LocateAll(both, nativeRoots(both)...)
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
	nested := filepath.Join(root, "appcache", "librarycache", "2", "4e1706e4", "library_hero.jpg")
	testfs.WriteAt(t, nested, []byte("jpg"))
	if got := s.HeroArt("2"); got != nested {
		t.Fatalf("nested hero = %q", got)
	}
}

func TestLaunchOptions(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		testfs.WriteAt(t, p, []byte(body))
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
	// A current client marks no account MostRecent, only each one's last sign-in, under a lower-case key.
	write("config/loginusers.vdf", `"users" { "76561198000000001" { "AccountName" "a" "timestamp" "100" } "76561198000000002" { "AccountName" "b" "timestamp" "200" } }`)
	if got, err := s.LaunchOptions("413150"); err != nil || !strings.Contains(got, "StardewModdingAPI.exe") {
		t.Fatalf("launch options by newest sign-in = %q, %v", got, err)
	}
	if got, err := s.LaunchOptions("1966720"); err != nil || got != "" {
		t.Fatalf("app without options = %q, %v", got, err)
	}
	if got, err := s.LaunchOptions("1"); err != nil || got != "" {
		t.Fatalf("unknown app = %q, %v", got, err)
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

func TestAppIDsThatLeaveTheLibraryAreRefused(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "outside.jpg"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := (Steam{Root: root}).HeroArt("../../outside.jpg"); got != "" {
		t.Fatalf("HeroArt followed a traversing app id to %q", got)
	}
	if got := BuildID(root, "../x"); got != "" {
		t.Fatalf("BuildID = %q", got)
	}
}

// A million levels (4 MB) overflowed the old parser's stack, a fatal error recover cannot catch.
func TestParseVDFRefusesWhatIsTooLargeOrTooDeep(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, body []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, body, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if _, err := parseVDF(write("deep.vdf", bytes.Repeat([]byte(`"a"{`), 1_000_000))); err == nil || !strings.Contains(err.Error(), "deeper") {
		t.Fatalf("deep: %v", err)
	}
	if _, err := parseVDF(write("big.vdf", bytes.Repeat([]byte(" "), maxTextVDFBytes+1))); err == nil || !strings.Contains(err.Error(), "larger") {
		t.Fatalf("big: %v", err)
	}
	// Braces and escaped quotes inside a string are text, not nesting.
	quoted := `"k" { "v" "` + strings.Repeat(`{ \" {`, 100) + `" }`
	if d, err := textVDFDepth([]byte(quoted)); err != nil || d != 1 {
		t.Fatalf("depth %d", d)
	}
	// The parser skips a // comment, so a quote or braces in one must not hide or add nesting.
	for name, body := range map[string][]byte{
		"quote": []byte("// \"\n" + strings.Repeat(`"a"{`, 40)),
		"brace": []byte("// " + strings.Repeat("{", 100) + "\n" + `"k" { "v" "1" }`),
	} {
		_, err := parseVDF(write(name+".vdf", body))
		if (name == "quote") != (err != nil && strings.Contains(err.Error(), "deeper")) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := parseVDF(write("ok.vdf", []byte(quoted))); err != nil {
		t.Fatalf("ok: %v", err)
	}
}

func TestLocateAllEmptyHomeIsTheUsersHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows finds Steam through the registry, not under the home folder")
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".steam", "steam", "steamapps"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())
	if got := LocateAll(""); len(got) != 1 || got[0].Root != filepath.Join(home, ".steam", "steam") {
		t.Fatalf("got %v", got)
	}
}

func FuzzTextVDF(f *testing.F) {
	f.Add([]byte("// \"\n" + strings.Repeat(`"a"{`, 40)))
	f.Add([]byte(`"k" { "v" "1" }`))
	f.Fuzz(func(t *testing.T, b []byte) {
		d, err := textVDFDepth(b)
		if err != nil || d > maxTextVDFDepth {
			return
		}
		if m, err := vdf.NewParser(bytes.NewReader(b)).Parse(); err == nil && m == nil {
			t.Fatal("nil map")
		}
	})
}
