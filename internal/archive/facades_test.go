package archive

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// ZipFunc walks a zip's entries without writing anything, and refuses a tarball whose bytes a zip reader would take.
func TestZipFuncWalksAZipAndRefusesAnotherFormat(t *testing.T) {
	t.Parallel()
	raw := testfs.ZipBytes(t, map[string]string{"Mod/manifest.json": "{}", "Mod/a.dll": "x"})
	var names []string
	err := ZipFunc(bytes.NewReader(raw), int64(len(raw)), Options{}, func(name string, isDir bool, r io.Reader) error {
		names = append(names, name)
		if name == "Mod/manifest.json" {
			b, err := io.ReadAll(r)
			if err != nil || string(b) != "{}" {
				t.Fatalf("entry body = %q, %v", b, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(slices.Sorted(slices.Values(names)), []string{"Mod/a.dll", "Mod/manifest.json"}) {
		t.Fatalf("entries = %v", names)
	}

	var tar bytes.Buffer
	tar.WriteString("ustar\x0000 not a zip")
	if err := ZipFunc(bytes.NewReader(tar.Bytes()), int64(tar.Len()), Options{}, func(string, bool, io.Reader) error { return nil }); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("a tarball was walked as a zip: %v", err)
	}
}

// A stream with no random access, which is how a download body arrives, extracts through its reader, and a body cut
// short fails rather than leaving a partial file behind as success.
func TestExtractReaderUnpacksAStreamAndFailsOneCutShort(t *testing.T) {
	t.Parallel()
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte("notes"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	root, err := os.OpenRoot(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := ExtractReader(bytes.NewReader(gz.Bytes()), "notes.txt.gz", root, Options{}); err != nil {
		t.Fatal(err)
	}
	if got, err := root.ReadFile("notes.txt"); err != nil || string(got) != "notes" {
		t.Fatalf("notes.txt = %q, %v", got, err)
	}

	cutRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cutRoot.Close() }()
	cut := gz.Bytes()[:gz.Len()-4]
	if err := ExtractReader(bytes.NewReader(cut), "notes.txt.gz", cutRoot, Options{}); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("a body cut short = %v, want ErrUnexpectedEOF", err)
	}
}

// An archive already in memory lists without a file, and closes the same way.
func TestListBytesWalksAnArchiveInMemory(t *testing.T) {
	t.Parallel()
	raw := testfs.ZipBytes(t, map[string]string{"a.txt": "aa", "b.txt": "bbb"})
	l, err := ListBytes(raw, "mem.zip", 0)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for {
		e, err := l.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		total += e.Size
	}
	if total != 5 {
		t.Fatalf("listed %d bytes, want 5", total)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := ListBytes([]byte("not a zip"), "mem.zip", 0); err == nil {
		t.Fatal("a non-archive listed")
	}
}

// DeclaredSize reads what the headers claim, without unpacking; a path that is not an archive or not there is an error.
func TestDeclaredSizeReadsTheHeaders(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	zipPath := testfs.WriteZip(t, filepath.Join(dir, "a.zip"), map[string]string{"a.txt": "aa", "b.txt": "bbb"})
	size, err := DeclaredSize(zipPath)
	if err != nil || size != 5 {
		t.Fatalf("DeclaredSize = %d, %v; want 5", size, err)
	}
	if _, err := DeclaredSize(filepath.Join(dir, "missing.zip")); err == nil {
		t.Fatal("a missing archive reported a size")
	}
	if _, err := DeclaredSize(testfs.WriteFile(t, dir, "text.zip", "plain text")); err == nil {
		t.Fatal("a text file reported a size")
	}
}

// A file-backed listing closes both the walk and the file, and one that cannot open the format closes the file it
// opened rather than leaking it.
func TestListOpensAndClosesAnArchiveFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	l, err := List(testfs.WriteZip(t, filepath.Join(dir, "a.zip"), map[string]string{"a.txt": "aa"}), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := l.Next(); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := List(testfs.WriteFile(t, dir, "text.zip", "plain text"), 0); !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("a text file listed: %v", err)
	}
}

// The dialog's pattern and the saved-name test cover every ending the reader takes, so a new format is offered and
// recognised without a second list.
func TestPickerPatternAndIsSavedCoverTheArchiveEndings(t *testing.T) {
	t.Parallel()
	pattern := PickerPattern()
	for _, ext := range []string{".zip", ".7z", ".rar", ".tar.gz"} {
		if !strings.Contains(pattern, "*"+ext) {
			t.Errorf("picker pattern %q lacks %s", pattern, ext)
		}
		if !IsSaved(SavedName("id", "mod"+ext)) {
			t.Errorf("SavedName of mod%s is not a saved name", ext)
		}
	}
	if IsSaved("mod.txt") {
		t.Error("a text file is not a saved download")
	}
}
