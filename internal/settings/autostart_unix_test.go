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
	s := Defaults()
	if err := ApplyKey(&s, "launchAtLogin", "true"); err != nil {
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
	if err := ApplyKey(&s, "launchAtLogin", "false"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Stat("mortar.desktop"); !os.IsNotExist(err) {
		t.Fatalf("expected desktop file gone, stat = %v", err)
	}
}
