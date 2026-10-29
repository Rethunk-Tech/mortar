package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func file() *fstest.MapFile { return &fstest.MapFile{Data: []byte("x")} }

func TestResolveFold(t *testing.T) {
	t.Parallel()
	both := fstest.MapFS{"Data/x.txt": file(), "data/x.txt": file(), "only/Sub/x.txt": file(), "k.png": file(), "K.png": file()}
	split := fstest.MapFS{"Data/x.txt": file(), "Data/sub/z.txt": file(), "data/y.txt": file()}
	collide := fstest.MapFS{"Data/a.txt": file(), "data/a.txt": file()}
	fileDir := fstest.MapFS{"a": file(), "A/b": file()}
	for _, tc := range []struct {
		name string
		fsys fs.FS
		rel  string
		want string
		err  error
	}{
		{"exact lower", both, "data/x.txt", "data/x.txt", nil},
		{"exact upper", both, "Data/x.txt", "Data/x.txt", nil},
		{"first variant", both, "DATA/x.txt", "Data/x.txt", nil},
		{"deeper variant", both, "ONLY/sub/x.txt", "only/Sub/x.txt", nil},
		{"sorted first of files", both, "k.PNG", "K.png", nil},
		{"missing", both, "missing/x.txt", "", fs.ErrNotExist},
		{"dotdot", both, "..", "", fs.ErrInvalid},
		{"absolute", both, "/Data", "", fs.ErrInvalid},
		{"root", both, ".", ".", nil},
		{"backtrack to x", split, "data/x.txt", "Data/x.txt", nil},
		{"backtrack to z", split, "data/sub/z.txt", "Data/sub/z.txt", nil},
		{"other variant", split, "Data/y.txt", "data/y.txt", nil},
		{"upper other variant", split, "DATA/y.txt", "data/y.txt", nil},
		{"collision exact", collide, "data/a.txt", "data/a.txt", nil},
		{"collision folded", collide, "DATA/a.txt", "Data/a.txt", nil},
		{"file used as directory", fileDir, "a/b", "A/b", nil},
		{"file used as directory, absent", fileDir, "a/c", "", fs.ErrNotExist},
	} {
		got, err := ResolveFold(tc.fsys, tc.rel)
		if !errors.Is(err, tc.err) || got != tc.want {
			t.Errorf("%s: ResolveFold(%q) = %q, %v; want %q, %v", tc.name, tc.rel, got, err, tc.want, tc.err)
		}
	}
}

func TestResolveFoldBudget(t *testing.T) {
	t.Parallel()
	// Both case variants at each of 8 levels, and the file only under the last chain tried: past 8 reads per element.
	fsys := fstest.MapFS{}
	for mask := range 1 << 8 {
		parts := make([]string, 8)
		for i := range parts {
			parts[i] = "a"
			if mask&(1<<i) != 0 {
				parts[i] = "A"
			}
		}
		fsys[strings.Join(parts, "/")+"/pad"] = file()
	}
	fsys[strings.Repeat("A/", 8)+"f"] = file()
	rel := strings.Repeat("a/", 8) + "f"
	if _, err := ResolveFold(fsys, rel); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a walk past its budget = %v, want fs.ErrNotExist", err)
	}
	if got, err := ResolveFold(fsys, strings.Repeat("A/", 8)+"f"); err != nil || got != strings.Repeat("A/", 8)+"f" {
		t.Fatalf("the exact chain = %q, %v", got, err)
	}
}

func TestResolveFoldRootRefusesALinkOut(t *testing.T) {
	t.Parallel()
	outside, inside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(inside, "Link")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	root, err := os.OpenRoot(inside)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if _, err := ResolveFold(root.FS(), "link/secret.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("through a link out of the root = %v, want fs.ErrNotExist", err)
	}
}

func FuzzResolveFold(f *testing.F) {
	fsys := fstest.MapFS{"Data/x.txt": file(), "Data/sub/z.txt": file(), "data/y.txt": file(), "data/Sub/w.txt": file()}
	for _, s := range []string{"data/x.txt", "Data/x.txt", "DATA/sub/z.txt", "data/y.txt", "Data/y.txt", "data/sub/w.txt", ".", "..", "", "/", "a//b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, rel string) {
		got, err := ResolveFold(fsys, rel)
		match := foldMatches(fsys, rel)
		if err != nil {
			if fs.ValidPath(rel) && match {
				t.Fatalf("%q has a folded match but resolved to %v", rel, err)
			}
			return
		}
		if !fs.ValidPath(got) || len(strings.Split(got, "/")) != len(strings.Split(rel, "/")) || !strings.EqualFold(got, rel) {
			t.Fatalf("ResolveFold(%q) = %q", rel, got)
		}
		if _, err := fs.Stat(fsys, got); err != nil {
			t.Fatalf("ResolveFold(%q) = %q, which does not exist", rel, got)
		}
		if _, err := fs.Stat(fsys, rel); err == nil && got != rel {
			t.Fatalf("%q exists but resolved to %q", rel, got)
		}
	})
}

// foldMatches reports whether some key of fsys, or a directory above one, matches rel element by element under folding.
func foldMatches(fsys fstest.MapFS, rel string) bool {
	if rel == "." {
		return true
	}
	want := strings.Split(rel, "/")
	for key := range fsys {
		have := strings.Split(key, "/")
		if len(have) < len(want) {
			continue
		}
		ok := true
		for i := range want {
			ok = ok && strings.EqualFold(have[i], want[i])
		}
		if ok {
			return true
		}
	}
	return false
}
