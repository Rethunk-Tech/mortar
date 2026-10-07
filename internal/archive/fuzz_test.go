package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/winname"
)

func FuzzCleanName(f *testing.F) {
	for _, s := range []string{"a/b.txt", "../x", `..\x`, "/abs", "a/./b", "a//b", "a\x00b", "CON", "a/NUL.txt", ""} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		got, err := cleanName(name)
		if err != nil {
			if !errors.Is(err, ErrTraversal) && !errors.Is(err, ErrUnsafeName) {
				t.Fatalf("untyped error for %q: %v", name, err)
			}
			return
		}
		if strings.HasPrefix(got, "/") || strings.Contains(got, `\`) || strings.ContainsRune(got, 0) {
			t.Fatalf("unsafe result %q from %q", got, name)
		}
		for s := range strings.SplitSeq(got, "/") {
			if s == ".." {
				t.Fatalf("escaping result %q from %q", got, name)
			}
		}
	})
}

// checkExtracted fails when anything was written outside dest (its sibling sentinel folder must stay empty and the
// parent must hold only the two), when dest holds a link or a special file, or when a name is one Windows would alter.
func checkExtracted(t *testing.T, parent, dest string) {
	t.Helper()
	top, err := os.ReadDir(parent)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 2 {
		t.Fatalf("extraction wrote beside its destination: %v", top)
	}
	if side, _ := os.ReadDir(filepath.Join(parent, "sentinel")); len(side) != 0 {
		t.Fatalf("extraction wrote into a sibling folder: %v", side)
	}
	seen := map[string]string{}
	err = filepath.WalkDir(dest, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dest {
			return err
		}
		if !d.Type().IsRegular() && !d.IsDir() {
			t.Fatalf("extracted a %v at %s", d.Type(), p)
		}
		if !winname.Valid(d.Name()) {
			t.Fatalf("extracted a name Windows would alter: %q", p)
		}
		rel, _ := filepath.Rel(dest, p)
		if prev, ok := seen[strings.ToLower(rel)]; ok {
			t.Fatalf("%q and %q collide on a case-insensitive disk", prev, rel)
		}
		seen[strings.ToLower(rel)] = rel
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func extractFuzzed(t *testing.T, archive []byte) {
	parent := t.TempDir()
	dest, src := filepath.Join(parent, "dest"), filepath.Join(t.TempDir(), "in")
	for _, d := range []string{dest, filepath.Join(parent, "sentinel")} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(src, archive, 0o600); err != nil {
		t.Fatal(err)
	}
	// Small caps keep a crafted bomb from filling the disk; extraction may fail, but never outside dest.
	// Listing and sizing read the same untrusted headers; they must fail, not panic.
	_, _ = PreviewArchive(src)
	_, _ = DeclaredSize(src)
	_ = extractWith(src, dest, options{MaxEntryBytes: 1 << 20, MaxTotalBytes: 4 << 20, MaxEntries: 200})
	checkExtracted(t, parent, dest)
}

// FuzzExtract feeds raw bytes, seeded with every format's fixtures, through the format detection and readers.
func FuzzExtract(f *testing.F) {
	seeds, _ := filepath.Glob("testdata/*")
	for _, p := range seeds {
		if b, err := fsx.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Add(zipOf(f, "../escape.txt", 0, []byte("x")))
	f.Add(zipOf(f, "link", fs.ModeSymlink|0o777, []byte("/etc/passwd")))
	f.Fuzz(func(t *testing.T, archive []byte) { extractFuzzed(t, archive) })
}

// FuzzExtractZipEntries writes well-formed zips whose two entries take fuzzed names and modes, so the name and kind
// checks see far more cases than mutated bytes reach.
func FuzzExtractZipEntries(f *testing.F) {
	for _, s := range []struct {
		a, b string
		mode uint32
	}{
		{"a/b.txt", "A/B.txt", 0},
		{"..\\x", "y", 0},
		{"/abs", "z", 0},
		{"dir/", "dir/f", uint32(fs.ModeDir)},
		{"link", "link/x", uint32(fs.ModeSymlink)},
		{"evil.txt:stream", "CON", 0},
		{"x.", "x", 0},
		{"a/../../b", "c", 0},
		{"pipe", "p", uint32(fs.ModeNamedPipe)},
		{"ｆｕｌｌ／ｗｉｄｔｈ", "\u202eexe.txt", 0},
	} {
		f.Add(s.a, s.b, s.mode, []byte("data"))
	}
	f.Fuzz(func(t *testing.T, a, b string, mode uint32, data []byte) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for i, name := range []string{a, b} {
			h := &zip.FileHeader{Name: name, Method: zip.Store}
			if i == 0 {
				h.SetMode(fs.FileMode(mode))
			}
			w, err := zw.CreateHeader(h)
			if err != nil {
				return
			}
			_, _ = w.Write(data)
		}
		if zw.Close() != nil {
			return
		}
		extractFuzzed(t, buf.Bytes())
	})
}

func zipOf(f *testing.F, name string, mode fs.FileMode, data []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: name, Method: zip.Store}
	h.SetMode(mode)
	w, err := zw.CreateHeader(h)
	if err != nil {
		f.Fatal(err)
	}
	_, _ = w.Write(data)
	if err := zw.Close(); err != nil {
		f.Fatal(err)
	}
	return buf.Bytes()
}

// FuzzExtractStreams seeds the detection and every stream reader with each compressed format, around a tar and bare,
// so mutations reach the decoders and the tar header parser.
func FuzzExtractStreams(f *testing.F) {
	seeds, _ := filepath.Glob("testdata/*.tar*")
	bare, _ := filepath.Glob("testdata/bare.*")
	for _, p := range append(seeds, bare...) {
		if b, err := fsx.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	f.Fuzz(func(t *testing.T, archive []byte) { extractFuzzed(t, archive) })
}
