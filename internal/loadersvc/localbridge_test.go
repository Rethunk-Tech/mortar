package loadersvc

import (
	"archive/zip"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestALocalBridgeBuildStandsInForAnUnpublishedRelease(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	t.Setenv("MORTAR_ENABLE_GAMES", "lethal-company")
	folder := t.TempDir()
	zipPath := filepath.Join(folder, "lethal-company.zip")
	f, err := fsx.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("plugins/MortarBepInExBridge.dll")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("bridge")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	items, profiles := testenv.Stores(t)
	// The manifest has no bridge for the game, as the shipped catalog has none for Lethal Company.
	svc := NewService(t.TempDir(), set, items, profiles, components.NewClient(http.DefaultClient))

	if b, err := svc.ensureBridge("lethal-company"); err != nil || b.Key != "" {
		t.Fatalf("without the variable: %+v, %v", b, err)
	}
	t.Setenv(localBridgeEnv, folder)
	b, err := svc.ensureBridge("lethal-company")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := fsx.SHA256(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if b.Key != store.BridgeKey("local", sum) {
		t.Fatalf("key = %q", b.Key)
	}
	dir, err := items.Path("lethal-company", b.Key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "plugins", "MortarBepInExBridge.dll")); err != nil {
		t.Fatal(err)
	}
}
