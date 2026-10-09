package dlwatch

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNexusFilename(t *testing.T) {
	inf, ok := ParseNexusFilename("ContentPatcher-1915-2-8-2-1700000000.zip")
	if !ok {
		t.Fatal("expected nexus name")
	}
	if inf.ModID != 1915 || inf.Name != "ContentPatcher" {
		t.Fatalf("got %+v", inf)
	}
	inf, ok = ParseNexusFilename("A_Mod-Name-42-1-0-1700000001.7z")
	if !ok || inf.ModID != 42 || inf.Name != "A Mod-Name" {
		t.Fatalf("got %+v ok=%v", inf, ok)
	}
	if _, ok := ParseNexusFilename("random.zip"); ok {
		t.Fatal("plain zip is not a nexus name")
	}
}

func TestPeekNameSkipsAnOversizedManifest(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "mod.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Mod/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write(append(bytes.Repeat([]byte(" "), 2*maxManifestBytes), `{"Name":"Big"}`...))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := PeekName(path); got != "" {
		t.Fatalf("PeekName = %q, want empty for a manifest over the cap", got)
	}
}
