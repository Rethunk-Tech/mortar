package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/bridge"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

func TestWriteBridgeConfigUsesPascalCaseAndMode(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, bridge.ModFolder)
	if err := WriteBridgeConfig(mod, BridgeConfig{OverlayEnabled: true, OverlayPort: 8123, OverlayToken: "secret-token"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(mod, "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	raw, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"OverlayEnabled", "OverlayPort", "OverlayToken"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s: %s", k, raw)
		}
	}
	if m["OverlayEnabled"] != true || m["OverlayPort"] != float64(8123) || m["OverlayToken"] != "secret-token" {
		t.Fatalf("config = %s", raw)
	}
}

func TestWriteBridgeConfigOffClearsEnabled(t *testing.T) {
	dir := t.TempDir()
	mod := filepath.Join(dir, bridge.ModFolder)
	if err := WriteBridgeConfig(mod, BridgeConfig{OverlayEnabled: true, OverlayPort: 9000, OverlayToken: "keep"}); err != nil {
		t.Fatal(err)
	}
	if err := WriteBridgeConfig(mod, BridgeConfig{OverlayEnabled: false, OverlayPort: 9000, OverlayToken: "keep"}); err != nil {
		t.Fatal(err)
	}
	raw, err := fsx.ReadFile(filepath.Join(mod, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c BridgeConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	if c.OverlayEnabled || c.OverlayPort != 9000 || c.OverlayToken != "keep" {
		t.Fatalf("%+v", c)
	}
}

func TestApplyToModsWritesEachBridgeFolder(t *testing.T) {
	mods := t.TempDir()
	a := filepath.Join(mods, "bridge-1.0.1", bridge.ModFolder)
	b := filepath.Join(mods, "other", "NotBridge")
	if err := os.MkdirAll(a, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ApplyToMods(mods, BridgeConfig{OverlayEnabled: false, OverlayPort: 8123, OverlayToken: "tok"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a, "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(b, "config.json")); err == nil {
		t.Fatal("wrote config into a non-bridge folder")
	}
}

func TestFileURLQueryNames(t *testing.T) {
	got := FileURL("/data/overlay/index.html", 8123, "ab&c")
	if !strings.HasPrefix(got, "file:///data/overlay/index.html?") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "port=8123") || !strings.Contains(got, "token=ab") {
		t.Fatalf("got %q", got)
	}
}

func TestWritePageLaysOutIndex(t *testing.T) {
	dir := t.TempDir()
	if err := WritePage(dir); err != nil {
		t.Fatal(err)
	}
	raw, err := fsx.ReadFile(PagePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "params.get(\"token\")") || !strings.Contains(string(raw), "params.get(\"port\")") {
		t.Fatalf("page missing query params: %s", raw[:min(200, len(raw))])
	}
}
