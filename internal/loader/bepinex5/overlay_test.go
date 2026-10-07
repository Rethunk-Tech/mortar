package bepinex5

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/overlay"
)

func TestArmOverlayWritesAndClearsTheBridgeConfig(t *testing.T) {
	var l loader.OverlayArmer = Loader{}
	dir := t.TempDir()
	if err := l.ArmOverlay(dir, true, 8123, "tok"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, filepath.FromSlash(overlay.ProfileConfigFile))
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(raw, &cfg); err != nil || cfg["OverlayEnabled"] != true || cfg["OverlayPort"] != float64(8123) || cfg["OverlayToken"] != "tok" {
		t.Fatalf("config %s (%v)", raw, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %v %v", info, err)
	}
	if err := l.ArmOverlay(dir, false, 8123, "tok"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("config kept after off: %v", err)
	}
	if err := l.ArmOverlay(dir, false, 8123, "tok"); err != nil {
		t.Fatalf("off twice: %v", err)
	}
}
