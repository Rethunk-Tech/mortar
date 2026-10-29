package archive

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

func zipAt(t *testing.T, files map[string]string) string {
	t.Helper()
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), "a.zip"), files)
}

func TestPreviewArchiveFindsManifestsAndFomod(t *testing.T) {
	t.Parallel()
	p, err := PreviewArchive(zipAt(t, map[string]string{
		"Mod/":                   "",
		"Mod/manifest.json":      `{"Name":"Mod","UniqueID":"A.Mod","Version":"1.2.0"}`,
		"fomod/ModuleConfig.xml": "<config/>",
		"Bad/manifest.json":      "not json",
	}))
	if err != nil {
		t.Fatal(err)
	}
	var dir bool
	for _, e := range p.Entries {
		if e.Path == "Mod" {
			dir = e.IsDir
		}
	}
	if len(p.Entries) != 4 || !dir {
		t.Fatalf("entries = %+v", p.Entries)
	}
	if len(p.Manifests) != 1 || p.Manifests[0] != (PreviewManifest{Folder: "Mod", ID: "smapi:A.Mod", Name: "Mod", Version: "1.2.0"}) {
		t.Fatalf("manifests = %+v", p.Manifests)
	}
	if !p.Fomod || p.Truncated || p.TotalSize == 0 {
		t.Fatalf("preview = %+v", p)
	}
}

func TestPreviewArchiveNamesAThunderstoreManifest(t *testing.T) {
	t.Parallel()
	p, err := PreviewArchive(zipAt(t, map[string]string{
		"manifest.json":   "\xef\xbb\xbf" + `{"name":"Pkg","version_number":"1.2.0"}`,
		"plugins/Pkg.dll": "x",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Manifests) != 1 || p.Manifests[0] != (PreviewManifest{Name: "Pkg", Version: "1.2.0"}) {
		t.Fatalf("manifests = %+v", p.Manifests)
	}
}

func TestPreviewArchiveTruncatesEntries(t *testing.T) {
	t.Parallel()
	files := map[string]string{}
	for i := range MaxPreviewEntries + 5 {
		files[fmt.Sprintf("f%d.txt", i)] = "x"
	}
	p, err := PreviewArchive(zipAt(t, files))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Entries) != MaxPreviewEntries || !p.Truncated || p.TotalSize != int64(len(files)) {
		t.Fatalf("entries=%d truncated=%v total=%d", len(p.Entries), p.Truncated, p.TotalSize)
	}
}

func TestFacadeRefusesAnUnsupportedFormat(t *testing.T) {
	t.Parallel()
	p := testfs.WriteFile(t, t.TempDir(), "x.zip", "plain text")
	if _, err := PreviewArchive(p); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("PreviewArchive: %v", err)
	}
	if err := Extract(p, t.TempDir()); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("Extract: %v", err)
	}
	if _, err := DeclaredSize(p); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("DeclaredSize: %v", err)
	}
}

func TestExtractAppliesWindowsNames(t *testing.T) {
	t.Parallel()
	err := Extract(zipAt(t, map[string]string{"CON.txt": "x"}), t.TempDir())
	if !errors.Is(err, ErrUnsafeName) {
		t.Fatalf("err = %v", err)
	}
}
