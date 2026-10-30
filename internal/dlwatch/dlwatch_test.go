package dlwatch

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestParseNexusFilenameShowsUnderscoresAsSpaces(t *testing.T) {
	inf, ok := ParseNexusFilename("A_Mod-Name-42-1-0-1700000001.7z")
	if !ok || inf.ModID != 42 || inf.Name != "A Mod-Name" {
		t.Fatalf("got %+v ok=%v", inf, ok)
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

func TestPeekNameReadsAThunderstoreManifest(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "pkg.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(`{"name":"Pkg","version_number":"1.2.0"}`))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := PeekName(path); got != "Pkg" {
		t.Fatalf("PeekName = %q, want Pkg", got)
	}
}

func TestPeekNameReadsATarGz(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "pkg.tar.gz")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte(`{"Name":"Tarred","UniqueID":"a.b"}`)
	if err := tw.WriteHeader(&tar.Header{Name: "Mod/manifest.json", Mode: 0o600, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := PeekName(path); got != "Tarred" {
		t.Fatalf("PeekName = %q, want Tarred from a .tar.gz", got)
	}
}
