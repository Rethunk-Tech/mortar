package stardew

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractBridgeLaysOutTheMod(t *testing.T) {
	dst := t.TempDir()
	if err := (Game{}).ExtractBridge(dst); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"manifest.json", "MortarSmapiBridge.dll"} {
		if _, err := os.Stat(filepath.Join(dst, "MortarSmapiBridge", name)); err != nil {
			t.Fatal(err)
		}
	}
	if (Game{}).BridgeVersion() != bridgeVersion {
		t.Fatal("version mismatch")
	}
}
