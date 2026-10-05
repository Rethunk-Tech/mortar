package archivesvc

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func writeZip(t *testing.T, dir, name, body string) string {
	t.Helper()
	return testfs.WriteZip(t, filepath.Join(dir, name), map[string]string{"Mod/manifest.json": body})
}

func TestDownloadsArchivesSkipsInstalledByKeyAndByName(t *testing.T) {
	dir := t.TempDir()
	stored := writeZip(t, dir, "stored.zip", `{"UniqueID":"A.One"}`)
	writeZip(t, dir, "named.zip", `{"UniqueID":"A.Two"}`)
	fresh := writeZip(t, dir, "Cool Mod-1234-1-0-1700000000.zip", `{"UniqueID":"A.Three"}`)
	older := time.Now().Add(-time.Hour)
	if err := os.Chtimes(stored, older, older); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum, err := fsx.SHA256(stored)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(Deps{
		Dirs: func() []string { return []string{dir} },
		Keys: func(string) ([]string, error) { return []string{store.LocalKey(sum)}, nil },
		Profiles: func(string) ([]profile.Profile, error) {
			return []profile.Profile{{Entries: []profile.Entry{{Source: profile.Source{Kind: profile.KindLocal, Name: "named.zip"}}}}}, nil
		},
		NexusMods: func() map[int]bool { return map[int]bool{1234: true} },
	})
	got, err := s.DownloadsArchives("stardew")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Path != fresh || !got[0].KnownNexus || got[0].ModID != 1234 || got[0].Size == 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestDownloadsArchivesMissingDirAndPreview(t *testing.T) {
	s := NewService(Deps{Dirs: func() []string { return []string{filepath.Join(t.TempDir(), "none")} }})
	if got, err := s.DownloadsArchives("stardew"); err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
	p, err := s.ArchivePreview(writeZip(t, t.TempDir(), "a.zip", `{"UniqueID":"A.One","Name":"One","Version":"1.0"}`))
	if err != nil || len(p.Manifests) != 1 || p.Manifests[0].ID != "smapi:A.One" {
		t.Fatalf("preview %+v, %v", p, err)
	}
}

func TestNewDownloadsOffersOnlyArrivalsAfterTheMark(t *testing.T) {
	dir := t.TempDir()
	old := writeZip(t, dir, "old.zip", `{"UniqueID":"A.Old"}`)
	if err := os.Chtimes(old, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var seen int64
	on := true
	s := NewService(Deps{
		Dirs:    func() []string { return []string{dir} },
		Offer:   func(string) bool { return on },
		Seen:    func(string) int64 { return seen },
		SetSeen: func(_ string, m int64) error { seen = m; return nil },
	})
	if got, err := s.NewDownloads("stardew"); err != nil || len(got) != 0 || seen == 0 {
		t.Fatalf("first call sets the mark only: %v %v %d", got, err, seen)
	}
	writeZip(t, dir, "new.zip", `{"UniqueID":"A.New"}`)
	writeZip(t, dir, "0123456789abcdef.zip", `{"UniqueID":"A.Queued"}`)
	got, err := s.NewDownloads("stardew")
	if err != nil || len(got) != 1 || got[0].Name != "new.zip" {
		t.Fatalf("got %+v %v", got, err)
	}
	if again, _ := s.NewDownloads("stardew"); len(again) != 0 {
		t.Fatalf("offered twice: %+v", again)
	}
	on = false
	writeZip(t, dir, "later.zip", `{"UniqueID":"A.Later"}`)
	if got, _ := s.NewDownloads("stardew"); len(got) != 0 {
		t.Fatalf("offer off: %+v", got)
	}
}

func TestDownloadsArchivesReusesCachedHashes(t *testing.T) {
	dir := t.TempDir()
	stored := writeZip(t, dir, "stored.zip", `{"UniqueID":"A.One"}`)
	sum, err := fsx.SHA256(stored)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(t.TempDir(), "hashes.json")
	s := NewService(Deps{
		Dirs:      func() []string { return []string{dir} },
		HashCache: cache,
		Keys:      func(string) ([]string, error) { return []string{store.LocalKey(sum)}, nil },
	})
	if got, err := s.DownloadsArchives("stardew"); err != nil || len(got) != 0 {
		t.Fatalf("got %+v, %v", got, err)
	}
	// Same size and mtime with different bytes: only a cache hit can still say "installed".
	info, _ := os.Stat(stored)
	if err := os.WriteFile(stored, bytes.Repeat([]byte("x"), int(info.Size())), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stored, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if got, err := s.DownloadsArchives("stardew"); err != nil || len(got) != 0 {
		t.Fatalf("cache not used: %+v, %v", got, err)
	}
	if err := os.Chtimes(stored, info.ModTime().Add(time.Second), info.ModTime().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.DownloadsArchives("stardew"); len(got) != 1 {
		t.Fatalf("changed file must be rehashed: %+v", got)
	}
}

func TestDownloadsArchivesJoinsFoldersOncePerPath(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	writeZip(t, a, "one.zip", `{"UniqueID":"A.One"}`)
	writeZip(t, b, "two.zip", `{"UniqueID":"A.Two"}`)
	s := NewService(Deps{Dirs: func() []string { return []string{a, b, a, "", filepath.Join(a, "missing")} }})
	got, err := s.DownloadsArchives("stardew")
	if err != nil || len(got) != 2 {
		t.Fatalf("got %+v %v", got, err)
	}
}
