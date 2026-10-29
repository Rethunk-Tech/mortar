package gamestore

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// The GamesConfig shape is a real Heroic 2.x file's.
func TestHeroicPrefixIsThePfxFolderOfAProtonPrefix(t *testing.T) {
	dir := t.TempDir()
	prefix := filepath.Join(dir, "Prefixes", "Game")
	for _, p := range []string{filepath.Join(prefix, "pfx", "drive_c"), filepath.Join(dir, "Wine", "drive_c"), filepath.Join(dir, "GamesConfig")} {
		if err := os.MkdirAll(p, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	write := func(app, body string) {
		if err := os.WriteFile(filepath.Join(dir, "GamesConfig", app+".json"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("proton1", `{"proton1":{"wineVersion":{"type":"proton"},"winePrefix":`+strconv.Quote(prefix)+`},"version":"v0"}`)
	write("wine1", `{"wine1":{"wineVersion":{"type":"wine"},"winePrefix":`+strconv.Quote(filepath.Join(dir, "Wine"))+`}}`)
	write("gone", `{"gone":{"winePrefix":`+strconv.Quote(filepath.Join(dir, "missing"))+`}}`)
	for app, want := range map[string]string{"proton1": filepath.Join(prefix, "pfx"), "wine1": filepath.Join(dir, "Wine"), "gone": "", "none": "", "../x": ""} {
		if got := HeroicPrefix(dir, app); got != want {
			t.Errorf("%s: %q, want %q", app, got, want)
		}
	}
}
