package store

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	_ "github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	return &Store{root: filepath.Join(t.TempDir(), "store")}
}

func buildZip(t *testing.T, files map[string]string) string {
	t.Helper()
	return testfs.WriteZip(t, filepath.Join(t.TempDir(), "mod.zip"), files)
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
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
	if got := names(t, filepath.Join(s.root, blobsDir)); len(got) != 1 || got[0] != strings.TrimPrefix(key, "local-") {
		t.Fatalf("blobs = %v", got)
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
	if got := names(t, filepath.Join(s.root, blobsDir)); len(got) != 0 {
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
	err := s.install("stardew", "smapi-1", filepath.Join(s.root, loadersDir, "stardew", "smapi", "1"), func(string) error { return syscall.ENOSPC }, func() int64 { return n })
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
	if err := s.AddDir("stardew", LoaderKey("smapi", "4.1.0"), src); err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", "smapi-4.1.0")
	if err != nil || dir != filepath.Join(s.root, "loaders", "stardew", "smapi", "4.1.0") {
		t.Fatalf("dir = %q, %v", dir, err)
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
	testfs.WriteFile(t, first, "mod.dll", "old")
	if err := s.AddDir("stardew", "local-item", first); err != nil {
		t.Fatal(err)
	}
	dir, err := s.Path("stardew", "local-item")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, CompleteMarker)); err != nil {
		t.Fatal(err)
	}
	second := t.TempDir()
	testfs.WriteFile(t, second, "mod.dll", "new")

	if err := s.AddDir("stardew", "local-item", second); err != nil {
		t.Fatal(err)
	}
	dir, err = s.Path("stardew", "local-item")
	if err != nil {
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

func TestItemsWithoutTheCompleteMarkerAreIncomplete(t *testing.T) {
	s := newStore(t)
	blob := strings.Repeat("a", 64)
	dir := filepath.Join(s.root, blobsDir, blob)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.saveIndex(index{"stardew": {"unmarked": {Blob: blob}}}); err != nil {
		t.Fatal(err)
	}
	testfs.WriteFile(t, dir, "mod.dll", "old")

	_, err := s.Path("stardew", "unmarked")
	if !errors.Is(err, ErrIncomplete) {
		t.Fatalf("Path = %v", err)
	}
	if completeItem(dir) {
		t.Fatal("incomplete item was marked complete")
	}
}

func TestCorruptIndexIsSetAsideForInstall(t *testing.T) {
	s := newStore(t)
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(s.indexPath(), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	testfs.WriteFile(t, src, "mod.dll", "mod")

	if err := s.AddDir("stardew", "local-corrupt-index", src); err != nil {
		t.Fatal(err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if r, ok := idx["stardew"]["local-corrupt-index"]; !ok || time.Since(r.Used) > time.Minute || r.Blob == "" {
		t.Fatalf("new index = %v", idx)
	}
	if _, err := os.Stat(s.indexPath() + ".corrupt"); err != nil {
		t.Fatalf("corrupt index was not kept: %v", err)
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
	idx := index{"stardew": {"smapi-1": {Used: old}, "smapi-2": {Used: old}, "smapi-3": {Used: recent}, "smapi-4": {Used: old}}}
	if err := s.saveIndex(idx); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := s.Collect(map[string][]string{"stardew": {"smapi-1"}}, now); err != nil {
		t.Fatal(err)
	}
	got := names(t, filepath.Join(s.root, loadersDir, "stardew", "smapi"))
	if len(got) != 2 || got[0] != "1" || got[1] != "3" {
		t.Fatalf("kept %v, want smapi-1 (referenced) and smapi-3 (recent)", got)
	}
	after, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if used := after["stardew"]["smapi-1"].Used; !used.Equal(now.UTC()) && used.Sub(now) > time.Second {
		t.Errorf("referenced item not refreshed: %v", used)
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
	if err := os.MkdirAll(filepath.Join(s.root, loadersDir, "stardew", "smapi", ".tmp-left"), 0o700); err != nil {
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
	kept := names(t, filepath.Join(s.root, loadersDir, "stardew", "smapi"))
	if len(kept) != 2 || kept[0] != ".tmp-left" || kept[1] != "1" {
		t.Fatalf("after Remove = %v", kept)
	}
}

func TestTouchAndCleanup(t *testing.T) {
	s := newStore(t)
	if err := s.AddDir("stardew", "smapi-1", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := s.saveIndex(index{"stardew": {"smapi-1": {Used: time.Unix(0, 0).UTC()}}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Touch("stardew", "smapi-1"); err != nil {
		t.Fatal(err)
	}
	if idx, _ := s.loadIndex(); time.Since(idx["stardew"]["smapi-1"].Used) > time.Minute {
		t.Errorf("touch did not update: %v", idx)
	}

	tmp := filepath.Join(s.root, loadersDir, "stardew", "smapi", tempPrefix+"123")
	if err := os.MkdirAll(filepath.Join(tmp, "x"), 0o750); err != nil {
		t.Fatal(err)
	}
	if removed, err := s.Cleanup(); err != nil || len(removed) != 1 || removed[0] != "loaders/stardew/smapi/"+tempPrefix+"123" {
		t.Fatalf("cleanup = %v, %v", removed, err)
	}
	if got := names(t, filepath.Join(s.root, loadersDir, "stardew", "smapi")); len(got) != 1 || got[0] != "1" {
		t.Fatalf("after cleanup: %v", got)
	}
	if _, err := (&Store{root: filepath.Join(t.TempDir(), "none")}).Cleanup(); err != nil {
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
	testfs.WriteFile(t, src, "a.txt", "a")
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

func TestCollectLeavesFoldersThatAreNotStoreItems(t *testing.T) {
	s := newStore(t)
	stray := filepath.Join(s.root, blobsDir, "Not A Blob")
	if err := os.MkdirAll(stray, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := s.Collect(nil, time.Now().Add(365*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stray); err != nil {
		t.Fatalf("a folder that is not a store item was removed: %v", err)
	}
	refs, err := s.Unreferenced(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("unreferenced = %v", refs)
	}
}

func TestLoaderOf(t *testing.T) {
	if id, v, ok := LoaderOf(LoaderKey("bepinex5", "5.4.2304")); !ok || id != "bepinex5" || v != "5.4.2304" {
		t.Fatalf("round trip = %q %q, %v", id, v, ok)
	}
	for _, key := range []string{"smapi-", "nexus-1-2", ""} {
		if _, _, ok := LoaderOf(key); ok {
			t.Fatalf("%q parsed", key)
		}
	}
}

func TestCollectWithRetentionOffRestartsTheClock(t *testing.T) {
	s := newStore(t)
	if err := s.AddDir("stardew", "smapi-1", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-90 * 24 * time.Hour).UTC()
	if err := s.saveIndex(index{"stardew": {"smapi-1": {Used: old}}}); err != nil {
		t.Fatal(err)
	}
	s.UnusedFor = -1
	if err := s.Collect(map[string][]string{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	s.UnusedFor = 30 * 24 * time.Hour
	if err := s.Collect(map[string][]string{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := names(t, filepath.Join(s.root, loadersDir, "stardew", "smapi")); len(got) != 1 {
		t.Fatalf("item deleted right after retention was turned on: %v", got)
	}
}

func TestKeysWithTheSameBytesShareOneBlobAndAreFoundBySource(t *testing.T) {
	s := newStore(t)
	p := buildZip(t, map[string]string{"Mod/manifest.json": "{}"})
	if err := s.AddArchiveKey("stardew", NexusKey(7, 9), p); err != nil {
		t.Fatal(err)
	}
	local, err := s.AddArchive("stardew", p)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(t, filepath.Join(s.root, blobsDir)); len(got) != 1 {
		t.Fatalf("blobs = %v", got)
	}
	if es, err := s.Entries(); err != nil || len(es) != 2 || es[0].Dir != es[1].Dir {
		t.Fatalf("entries = %+v, %v", es, err)
	}
	if key, ok := s.Find("stardew", "nexus", "7", "9"); !ok || key != NexusKey(7, 9) {
		t.Fatalf("Find = %q, %v", key, ok)
	}
	if err := s.Describe("stardew", local, "github", "o/r", "v1", ""); err != nil {
		t.Fatal(err)
	}
	if key, ok := s.Find("stardew", "github", "o/r", "v1"); !ok || key != local {
		t.Fatalf("Find after Describe = %q, %v", key, ok)
	}
	if err := s.Remove([]Ref{{Game: "stardew", Key: NexusKey(7, 9)}}); err != nil {
		t.Fatal(err)
	}
	if got := names(t, filepath.Join(s.root, blobsDir)); len(got) != 1 {
		t.Fatalf("blob went with its first key: %v", got)
	}
	if err := s.Remove([]Ref{{Game: "stardew", Key: local}}); err != nil {
		t.Fatal(err)
	}
	if got := names(t, filepath.Join(s.root, blobsDir)); len(got) != 0 {
		t.Fatalf("blob outlived its last key: %v", got)
	}
}

// The decoded index is reused between reads, so it must follow a write from another Store on the same folder and
// must not pick up a change a caller makes to its own copy.
func TestIndexReuseFollowsTheFile(t *testing.T) {
	s := newStore(t)
	other := &Store{root: s.root}
	if err := s.AddDir("stardew", "smapi-1", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Path("stardew", "smapi-1"); err != nil {
		t.Fatal(err)
	}
	if err := other.AddDir("stardew", "smapi-2", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Path("stardew", "smapi-2"); err != nil {
		t.Fatalf("a key another store added: %v", err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	delete(idx["stardew"], "smapi-1")
	if _, err := s.Path("stardew", "smapi-1"); err != nil {
		t.Fatalf("a caller's change to its copy reached the shared index: %v", err)
	}
}

// BenchmarkPath resolves one key against an index the size of a large profile's store (800 items).
func BenchmarkPath(b *testing.B) {
	s := &Store{root: filepath.Join(b.TempDir(), "store")}
	recs := map[string]record{}
	for i := range 800 {
		recs["local-"+strings.Repeat("0", 60)+strconv.Itoa(1000+i)] = record{Source: "local", Used: time.Now()}
	}
	if err := s.AddDir("stardew", "smapi-1", b.TempDir()); err != nil {
		b.Fatal(err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		b.Fatal(err)
	}
	maps.Copy(idx["stardew"], recs)
	if err := s.saveIndex(idx); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := s.Path("stardew", "smapi-1"); err != nil {
			b.Fatal(err)
		}
	}
}

// An item's size is measured when it is added, and one added before that is filled in once and kept.
func TestEntriesCarryRecordedSizes(t *testing.T) {
	s := newStore(t)
	src := t.TempDir()
	testfs.WriteFile(t, src, "mod.dll", "12345")
	if err := s.AddDir("stardew", "local-sized", src); err != nil {
		t.Fatal(err)
	}
	entries, err := s.Entries()
	if err != nil || len(entries) != 1 || entries[0].Size != 5 {
		t.Fatalf("entries = %+v, %v", entries, err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	r := idx["stardew"]["local-sized"]
	r.Size = 0
	idx["stardew"]["local-sized"] = r
	if err := s.saveIndex(idx); err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSizes(map[string]map[string]int64{"stardew": {"local-sized": 5}}); err != nil {
		t.Fatal(err)
	}
	if entries, _ := s.Entries(); entries[0].Size != 5 {
		t.Fatalf("recorded size = %d", entries[0].Size)
	}
}

// Cleanup and the health check list the same unused items; an unreferenced entry whose folder is gone is in neither
// list and is pruned silently, while a referenced one stays for its profile to fetch again.
func TestUnreferencedMatchesReportAndDanglingEntriesArePruned(t *testing.T) {
	s := newStore(t)
	for _, k := range []string{"local-a", "local-b", "local-c"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "f"), []byte(k), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := s.AddDir("stardew", k, dir); err != nil {
			t.Fatal(err)
		}
	}
	for _, k := range []string{"local-b", "local-c"} {
		dir, err := s.Dir("stardew", k)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}
	keep := map[string][]string{"stardew": {"local-c"}}
	refs, err := s.Unreferenced(keep)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := s.Report(keep)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Key != "local-a" || len(rep["stardew"].Unused) != 1 || rep["stardew"].Unused[0].Key != "local-a" {
		t.Fatalf("cleanup lists %+v, the report %+v", refs, rep["stardew"].Unused)
	}
	if err := s.PruneDangling(keep); err != nil {
		t.Fatal(err)
	}
	idx, err := s.loadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx["stardew"]["local-b"]; ok || len(idx["stardew"]) != 2 {
		t.Fatalf("index after prune = %v", idx["stardew"])
	}
}
