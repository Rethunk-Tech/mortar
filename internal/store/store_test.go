package store

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/archive"
	"github.com/Rethunk-AI/mortar/internal/datadir"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{root: filepath.Join(t.TempDir(), "store")}
}

func buildZip(t *testing.T, files map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mod.zip")
	f, err := fsx.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestAddArchiveIsIdempotent(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{"Mod/manifest.json": "{}"})
	key, err := s.AddArchive("stardew", p)
	if err != nil || !strings.HasPrefix(key, "local-") || len(key) != len("local-")+64 {
		t.Fatalf("key = %q, %v", key, err)
	}
	dir, err := s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "Mod", "manifest.json")); err != nil || string(b) != "{}" {
		t.Fatalf("extracted = %q, %v", b, err)
	}
	again, err := s.AddArchive("stardew", p)
	if err != nil || again != key {
		t.Fatalf("re-add = %q, %v", again, err)
	}
	if got := names(t, filepath.Join(s.root, "stardew")); len(got) != 1 {
		t.Fatalf("game dir = %v", got)
	}
}

func TestLoadIndexRejectsSymlink(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(outside, []byte(`{"stardew":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, s.indexPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.loadIndex(); err == nil {
		t.Fatal("symlink index was read")
	}
}

func TestKeysDoesNotStampCompleteMarkerVersion(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{"Mod/manifest.json": "{}"})
	if _, err := s.AddArchive("stardew", p); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Keys("stardew"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(s.indexPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "complete-marker-v1") {
		t.Fatal("index still stamps complete-marker-v1")
	}
}

func TestAddArchiveFailureLeavesNothing(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{"ok.txt": "x", "../evil.txt": "y"})
	_, err := s.AddArchive("stardew", p)
	var ae *archive.Error
	if !errors.As(err, &ae) || !errors.Is(err, archive.ErrTraversal) {
		t.Fatalf("err = %v", err)
	}
	if got := names(t, filepath.Join(s.root, "stardew")); len(got) != 0 {
		t.Fatalf("left behind: %v", got)
	}
}

func TestDiskFullMessage(t *testing.T) {
	s := newStore(t)
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "a"), make([]byte, 3<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	n, _ := datadir.Size(src)
	err := s.install("stardew", "smapi-1", func(string) error { return syscall.ENOSPC }, func() int64 { return n })
	if !errors.Is(err, syscall.ENOSPC) || !strings.Contains(err.Error(), "needs about 4 MB") {
		t.Fatalf("err = %v", err)
	}
}

func TestKeyAndGameValidation(t *testing.T) {
	s := newStore(t)
	for _, key := range []string{"", ".hidden", "..", "a/b", `a\b`, "../x", "UPPER", "a b", "a:b"} {
		if _, err := s.Path("stardew", key); err == nil || errors.Is(err, ErrNotFound) {
			t.Errorf("Path(%q) = %v, want validation error", key, err)
		}
		if err := s.AddDir("stardew", key, t.TempDir()); err == nil {
			t.Errorf("AddDir(%q) succeeded", key)
		}
	}
	if _, err := s.Path("nope", "smapi-4.0.0"); err == nil {
		t.Error("unknown game accepted")
	}
	if _, err := s.Path("stardew", "smapi-4.0.0"); !errors.Is(err, ErrNotFound) {
		t.Errorf("absent key = %v", err)
	}
}

func TestAddDir(t *testing.T) {
	s := newStore(t)
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "ConsoleCommands"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(src, "ConsoleCommands", "manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDir("stardew", SMAPIKey("4.1.0"), src); err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", "smapi-4.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ConsoleCommands", "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDir("stardew", "smapi-4.1.0", src); err != nil {
		t.Fatalf("re-add: %v", err)
	}
}

func TestIncompleteItemIsReinstalled(t *testing.T) {
	s := newStore(t)
	first := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(first, "mod.dll"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDir("stardew", "local-item", first); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.root, "stardew", "local-item")
	if err := os.Remove(filepath.Join(dir, completeMarker)); err != nil {
		t.Fatal(err)
	}
	second := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(second, "mod.dll"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.AddDir("stardew", "local-item", second); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(dir, "mod.dll"))
	if err != nil || string(got) != "new" {
		t.Fatalf("reinstalled item = %q, %v", got, err)
	}
	if !completeItem(dir) {
		t.Fatal("reinstalled item has no completion marker")
	}
}

func TestLegacyItemsWithoutArchiveAreIncomplete(t *testing.T) {
	s := newStore(t)
	dir := filepath.Join(s.root, "stardew", "legacy")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(dir, "mod.dll"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := s.Path("stardew", "legacy")
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("Path = %v", err)
	}
	if completeItem(dir) {
		t.Fatal("incomplete item was marked complete")
	}
}

func TestIncompleteItemIsReextractedFromSourceArchive(t *testing.T) {
	s := newStore(t)
	zipPath := buildZip(t, map[string]string{"Mod/manifest.json": "{}"})
	key, err := s.AddArchive("stardew", zipPath)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.root, "stardew", key)
	if err := os.Remove(filepath.Join(dir, completeMarker)); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "Mod")); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(s.root, "stardew", key+".zip")
	if err := datadirCopy(t, zipPath, src); err != nil {
		t.Fatal(err)
	}
	if err := s.RepairIncomplete(); err != nil {
		t.Fatal(err)
	}
	if !completeItem(dir) {
		t.Fatal("item was not re-extracted")
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "Mod", "manifest.json")); err != nil || string(b) != "{}" {
		t.Fatalf("re-extracted = %q, %v", b, err)
	}
}

func datadirCopy(t *testing.T, src, dst string) error {
	t.Helper()
	b, err := fsx.ReadFile(src)
	if err != nil {
		return err
	}
	return fsx.WriteFile(dst, b, 0o600)
}

func TestAddDirVerifiedChecksLocalKey(t *testing.T) {
	s := newStore(t)
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "mod.dll"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, err := hashDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddDirVerified("stardew", key, src); err != nil {
		t.Fatal(err)
	}
	if err := s.AddDirVerified("stardew", LocalKey("wrong"), src); err == nil {
		t.Fatal("mismatched local key was accepted")
	}
}

func TestCorruptIndexIsRebuiltForInstall(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(s.indexPath(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "mod.dll"), []byte("mod"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := s.AddDir("stardew", "local-corrupt-index", src); err != nil {
		t.Fatal(err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	last, ok := idx["stardew"]["local-corrupt-index"]
	if !ok || time.Since(last) > time.Minute {
		t.Fatalf("rebuilt index = %v", idx)
	}
}

func TestCollect(t *testing.T) {
	s := newStore(t)
	for _, k := range []string{"smapi-1", "smapi-2", "smapi-3", "smapi-4"} {
		if err := s.AddDir("stardew", k, t.TempDir()); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-40 * 24 * time.Hour).UTC()
	recent := time.Now().Add(-5 * 24 * time.Hour).UTC()
	idx := index{"stardew": {"smapi-1": old, "smapi-2": old, "smapi-3": recent, "smapi-4": old}}
	if err := s.saveIndex(idx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.Collect(map[string][]string{"stardew": {"smapi-1"}}, now); err != nil {
		t.Fatal(err)
	}
	got := names(t, filepath.Join(s.root, "stardew"))
	if len(got) != 2 || got[0] != "smapi-1" || got[1] != "smapi-3" {
		t.Fatalf("kept %v, want smapi-1 (referenced) and smapi-3 (recent)", got)
	}
	after, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if !after["stardew"]["smapi-1"].Equal(now.UTC()) && after["stardew"]["smapi-1"].Sub(now) > time.Second {
		t.Errorf("referenced item not refreshed: %v", after["stardew"]["smapi-1"])
	}
	if _, ok := after["stardew"]["smapi-2"]; ok || len(after["stardew"]) != 2 {
		t.Errorf("index = %v", after)
	}
}

func TestUnreferencedKeepsReferenced(t *testing.T) {
	s := newStore(t)
	for _, k := range []string{"smapi-1", "smapi-2"} {
		if err := s.AddDir("stardew", k, t.TempDir()); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(s.root, "stardew", ".tmp-left"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := s.Unreferenced(map[string][]string{"stardew": {"smapi-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Game != "stardew" || got[0].Key != "smapi-2" {
		t.Fatalf("Unreferenced = %+v", got)
	}
	if err := s.Remove(got); err != nil {
		t.Fatal(err)
	}
	kept := names(t, filepath.Join(s.root, "stardew"))
	if len(kept) != 2 || kept[0] != ".tmp-left" || kept[1] != "smapi-1" {
		t.Fatalf("after Remove = %v", kept)
	}
}

func TestTouchAndCleanup(t *testing.T) {
	s := newStore(t)
	if err := s.AddDir("stardew", "smapi-1", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := s.saveIndex(index{"stardew": {"smapi-1": time.Unix(0, 0).UTC()}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Touch("stardew", "smapi-1"); err != nil {
		t.Fatal(err)
	}
	if idx, _ := s.loadIndex(); time.Since(idx["stardew"]["smapi-1"]) > time.Minute {
		t.Errorf("touch did not update: %v", idx)
	}

	tmp := filepath.Join(s.root, "stardew", tempPrefix+"123")
	if err := os.MkdirAll(filepath.Join(tmp, "x"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := s.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if got := names(t, filepath.Join(s.root, "stardew")); len(got) != 1 || got[0] != "smapi-1" {
		t.Fatalf("after cleanup: %v", got)
	}
	if err := (&Store{root: filepath.Join(t.TempDir(), "none")}).Cleanup(); err != nil {
		t.Fatalf("cleanup on missing root: %v", err)
	}
}

func TestDeclaredSize(t *testing.T) {
	p := buildZip(t, map[string]string{"a": "12345", "b/c": "678"})
	if n, err := archive.DeclaredSize(p); err != nil || n != 8 {
		t.Fatalf("size = %d, %v", n, err)
	}
}

func TestAddHashedDirCopiesInTreeSymlinksAndRejectsEscapes(t *testing.T) {
	s := newStore(t)
	src := t.TempDir()
	if err := fsx.WriteFile(filepath.Join(src, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src, "a.txt"), filepath.Join(src, "b.txt")); err != nil {
		t.Fatal(err)
	}
	key, err := s.AddHashedDir("stardew", src)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(filepath.Join(dir, "b.txt"))
	if err != nil || string(got) != "a" {
		t.Fatalf("hashed symlink = %q, %v", got, err)
	}

	escaped := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret")
	if err := fsx.WriteFile(outside, []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(escaped, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddHashedDir("stardew", escaped); err == nil {
		t.Fatal("escaped symlink hashed")
	}
}

func TestAddArchiveStripsJunkFolders(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{
		"__MACOSX/foo/manifest.json": `{"UniqueID":"Junk.A"}`,
		"Good/manifest.json":         `{"UniqueID":"Good.A"}`,
		"thumbs/Thumbs.db":           "x",
	})
	key, err := s.AddArchive("stardew", p)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	got := names(t, dir)
	if strings.Contains(strings.Join(got, ","), "__MACOSX") || strings.Contains(strings.Join(got, ","), "thumbs") {
		t.Fatalf("junk left in %v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "Good", "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestPathUsesStoredRootWhenPresent(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{"Outer/Mod/manifest.json": `{"UniqueID":"M.A"}`, "readme.txt": "x"})
	key, err := s.AddArchive("stardew", p)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetRoot("stardew", key, "Outer/Mod"); err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "Mod" {
		t.Fatalf("path = %s", dir)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	dir, err = s.Path("stardew", key)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) == "Mod" {
		t.Fatalf("missing root still used %s", dir)
	}
}
