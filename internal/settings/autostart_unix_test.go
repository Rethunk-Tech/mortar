//go:build unix

package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLaunchAtLoginWritesAutostartFile(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	svc := NewService(mustOpen(t))
	if err := svc.SetByKey("launchAtLogin", "true", ""); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(filepath.Join(cfg, "autostart"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	body, err := root.ReadFile("mortar.desktop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Type=Application") || !strings.Contains(string(body), "Name=Mortar") {
		t.Fatalf("desktop file = %s", body)
	}
	if err := svc.SetByKey("launchAtLogin", "false", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Stat("mortar.desktop"); !os.IsNotExist(err) {
		t.Fatalf("expected desktop file gone, stat = %v", err)
	}
}

// A second Store opened before the set holds a stale snapshot; its later save must build on the accepted
// launchAtLogin, and the set must have written the autostart entry.
func TestQueuedLaunchAtLoginSurvivesSecondOpener(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	dir := filepath.Join(data, "mortar")
	app, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	other, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if err := NewService(app).SetByKey("launchAtLogin", "true", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Update(func(v *Settings) { v.LastGame = "stardew" }); err != nil {
		t.Fatal(err)
	}
	fresh, err := OpenIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.Get().LaunchAtLogin {
		t.Fatal("launchAtLogin lost by the second writer")
	}
	if _, err := os.Stat(filepath.Join(cfg, "autostart", "mortar.desktop")); err != nil {
		t.Fatalf("autostart entry: %v", err)
	}
}

func mustOpen(t *testing.T) *Store {
	t.Helper()
	s, err := OpenIn(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
