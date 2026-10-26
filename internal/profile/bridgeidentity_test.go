package profile

import (
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// The catalog's bridge is named by the catalog, not by its manifest: Thunderstore's manifest has no namespace.
func TestTheCatalogBridgeIsNamedWithoutAManifestNamespace(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	src := t.TempDir()
	testfs.WriteFile(t, src, "manifest.json", `{"name":"MortarBepInExBridge","version_number":"0.2.4","website_url":"","description":"","dependencies":[]}`)
	testfs.WriteFile(t, src, filepath.Join("plugins", "MortarBepInExBridge.dll"), "MZ")
	key := store.BridgeKey("0.2.4", "abc123")
	if err := addAnyDir(t, e.items, "lethal-company", key, src); err != nil {
		t.Fatal(err)
	}
	mods, ok, err := e.packageMods("lethal-company", key, "")
	if err != nil || !ok || len(mods) != 1 || mods[0].ID != "thunderstore:Rethunk-MortarBepInExBridge" {
		t.Fatalf("mods = %+v, %v, %v", mods, ok, err)
	}
}
