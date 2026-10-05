package archive

import (
	"fmt"
	"testing"
)

func TestPreviewArchiveZipFindsManifestsAndFomod(t *testing.T) {
	p, err := PreviewArchive(buildZip(t,
		zentry{name: "Mod/"},
		zentry{name: "Mod/manifest.json", body: `{"Name":"Mod","UniqueID":"A.Mod","Version":"1.2.0"}`},
		zentry{name: "fomod/ModuleConfig.xml", body: "<config/>"},
		zentry{name: "Bad/manifest.json", body: "not json"},
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Entries) != 4 || !p.Entries[0].IsDir || p.Entries[0].Path != "Mod" {
		t.Fatalf("entries = %+v", p.Entries)
	}
	if len(p.Manifests) != 1 || p.Manifests[0] != (PreviewManifest{Folder: "Mod", ID: "smapi:A.Mod", Name: "Mod", Version: "1.2.0"}) {
		t.Fatalf("manifests = %+v", p.Manifests)
	}
	if !p.Fomod || p.Truncated || p.TotalSize == 0 {
		t.Fatalf("preview = %+v", p)
	}
}

func TestPreviewArchiveTruncatesEntries(t *testing.T) {
	entries := make([]zentry, MaxPreviewEntries+5)
	for i := range entries {
		entries[i] = zentry{name: fmt.Sprintf("f%d.txt", i), body: "x"}
	}
	p, err := PreviewArchive(buildZip(t, entries...))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Entries) != MaxPreviewEntries || !p.Truncated || p.TotalSize != int64(len(entries)) {
		t.Fatalf("entries=%d truncated=%v total=%d", len(p.Entries), p.Truncated, p.TotalSize)
	}
}

func TestPreviewArchiveSevenZipRarAndUnsupported(t *testing.T) {
	for _, path := range []string{"testdata/valid.7z", "testdata/real.rar"} {
		p, err := PreviewArchive(path)
		if err != nil || len(p.Entries) == 0 {
			t.Fatalf("%s: %+v, %v", path, p, err)
		}
	}
	if _, err := PreviewArchive(writeTemp(t, "x.zip", []byte("plain text"))); err == nil {
		t.Fatal("want unsupported format error")
	}
}
