package archive

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
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

func TestPreviewArchiveRefusesANonArchive(t *testing.T) {
	t.Parallel()
	p := testfs.WriteFile(t, t.TempDir(), "x.zip", "plain text")
	if _, err := PreviewArchive(p); err == nil {
		t.Fatal("a text file previewed")
	}
}

func TestFacadeRefusesAnUnsupportedFormat(t *testing.T) {
	t.Parallel()
	p := testfs.WriteFile(t, t.TempDir(), "x.zip", "plain text")
	if err := Extract(p, t.TempDir()); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("Extract: %v", err)
	}
}

// A saved bare stream's file takes its original name, not the queue id SavedName put before it, and a saved empty
// tar unpacks to nothing.
func TestExtractNamesFromTheOriginalName(t *testing.T) {
	t.Parallel()
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte("notes"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	stream := filepath.Join(dir, SavedName("0a1b2c", "notes.txt.gz"))
	empty := filepath.Join(dir, SavedName("0d1e2f", "x.tar.gz"))
	for p, b := range map[string][]byte{stream: gz.Bytes(), empty: make([]byte, 1024)} {
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dest := t.TempDir()
	if err := Extract(stream, dest); err != nil {
		t.Fatal(err)
	}
	if got, err := safefs.ReadFile(filepath.Join(dest, "notes.txt")); err != nil || string(got) != "notes" {
		t.Fatalf("notes.txt = %q, %v", got, err)
	}
	dest = t.TempDir()
	if err := Extract(empty, dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadDir(dest); len(got) != 0 {
		t.Fatalf("an empty tar unpacked %d entries", len(got))
	}
}

// A missing archive, a destination that is not there and a handle already closed each fail as the file system
// reports it, rather than as a panic or a silent success.
func TestExtractReportsTheFileSystemError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := zipAt(t, map[string]string{"a.txt": "aa"})
	if err := Extract(filepath.Join(dir, "missing.zip"), t.TempDir()); err == nil {
		t.Fatal("a missing archive extracted")
	}
	if err := Extract(zipPath, filepath.Join(dir, "no-such-dest")); err == nil {
		t.Fatal("a missing destination extracted")
	}
	if _, err := List(filepath.Join(dir, "missing.zip"), 0); err == nil {
		t.Fatal("a missing archive listed")
	}
	f, err := safefs.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ExtractOpened(f, zipPath, t.TempDir()); err == nil {
		t.Fatal("a closed handle extracted")
	}
}

func TestExtractOpenedReadsTheFileItHoldsNotThePath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows refuses to replace a file another handle holds open, so the swap this guards against cannot happen")
	}
	held := zipAt(t, map[string]string{"held.txt": "held"})
	other := zipAt(t, map[string]string{"swapped.txt": "swapped"})
	f, err := safefs.Open(held)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := os.Rename(other, held); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractOpened(f, held, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "held.txt")); err != nil {
		t.Fatalf("the held archive was not the one extracted: %v", err)
	}
}
