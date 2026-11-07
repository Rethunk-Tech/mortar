package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestUnityNamesComeFromAppInfo(t *testing.T) {
	for name, tc := range map[string]struct {
		info          string
		company, prod string
		ok            bool
	}{
		"plain":     {"ZeekerssRBLX\nLethal Company\n", "ZeekerssRBLX", "Lethal Company", true},
		"crlf":      {"Pugstorm\r\nCore Keeper\r\n", "Pugstorm", "Core Keeper", true},
		"one line":  {"Only\n", "", "", false},
		"traversal": {"..\nx\n", "", "", false},
		"slash":     {"a/b\nx\n", "", "", false},
	} {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "Game_Data"), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Game_Data", "app.info"), []byte(tc.info), 0o600); err != nil {
			t.Fatal(err)
		}
		c, p, ok := unityNames(dir)
		if c != tc.company || p != tc.prod || ok != tc.ok {
			t.Errorf("%s: got %q %q %v", name, c, p, ok)
		}
	}
	if _, _, ok := unityNames(t.TempDir()); ok {
		t.Error("a folder without app.info is not a Unity build")
	}
}

// Whatever the catalog names for a game, DataDirs never hands the sandbox the home folder, the root or a relative path.
func TestDataDirsOfEveryCatalogGameAreNarrow(t *testing.T) {
	home := t.TempDir()
	for _, g := range components.Games() {
		for _, d := range DataDirs(home, settings.Defaults(), g.ID, "") {
			if d == home || d == "/" || !filepath.IsAbs(d) {
				t.Errorf("%s: %q is too broad to bind", g.ID, d)
			}
		}
	}
}
