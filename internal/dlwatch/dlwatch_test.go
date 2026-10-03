package dlwatch

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseNexusFilename(t *testing.T) {
	inf, ok := ParseNexusFilename("ContentPatcher-1915-2-8-2-1700000000.zip")
	if !ok {
		t.Fatal("expected nexus name")
	}
	if inf.ModID != 1915 || inf.Name != "ContentPatcher" {
		t.Fatalf("got %+v", inf)
	}
	inf, ok = ParseNexusFilename("A_Mod-Name-42-1-0-1700000001.7z")
	if !ok || inf.ModID != 42 || inf.Name != "A Mod-Name" {
		t.Fatalf("got %+v ok=%v", inf, ok)
	}
	if _, ok := ParseNexusFilename("random.zip"); ok {
		t.Fatal("plain zip is not a nexus name")
	}
}

func TestNoticeIgnoreAndInstallIndex(t *testing.T) {
	s := &Service{deps: Deps{
		Current: func() (string, string, string, error) { return "stardew", "p1", "Main", nil },
	}}
	path := filepath.Join(t.TempDir(), "ContentPatcher-1915-2-8-2-1700000000.zip")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	it, err := s.notice(path)
	if err != nil || it.State != StateAsked || it.ModID != 1915 || it.N != 1 {
		t.Fatalf("notice: %+v err=%v", it, err)
	}
	if err := s.Ignore(path); err != nil {
		t.Fatal(err)
	}
	if s.List()[0].State != StateIgnored {
		t.Fatal(s.List()[0].State)
	}
}

func TestSnapshotIgnoresExistingAndPartial(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "already-1-1-0-1700000000.zip")
	if err := os.WriteFile(existing, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(dir, "incoming-2-1-0-1700000001.zip")
	if err := os.WriteFile(partial, []byte("xx"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(partial+".crdownload", []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_700_000_000, 0)
	f := &Folder{Dir: dir, Stable: 2 * time.Second, Now: func() time.Time { return now }}
	if err := f.Snapshot(); err != nil {
		t.Fatal(err)
	}
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("existing files: %v", got)
	}

	_ = os.Remove(partial + ".crdownload")
	now = now.Add(time.Minute)
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("still ignored after partial gone: %v", got)
	}

	fresh := filepath.Join(dir, "FreshMod-9-1-0-1700000002.zip")
	if err := os.WriteFile(fresh, []byte("ab"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("new file must wait for stable size: %v", got)
	}
	now = now.Add(2 * time.Second)
	got := f.Poll()
	if len(got) != 1 || got[0] != fresh {
		t.Fatalf("expected fresh archive, got %v", got)
	}
}

func TestStabilityWaitsForSizeAndPartSibling(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(1_700_000_000, 0)
	f := &Folder{Dir: dir, Stable: 2 * time.Second, Now: func() time.Time { return now }}
	if err := f.Snapshot(); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "Grow-3-1-0-1700000003.zip")
	if err := os.WriteFile(path, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("first sight: %v", got)
	}
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("size changed: %v", got)
	}
	if err := os.WriteFile(path+".part", []byte("p"), 0o600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("part sibling: %v", got)
	}
	_ = os.Remove(path + ".part")
	now = now.Add(time.Second)
	if got := f.Poll(); len(got) != 0 {
		t.Fatalf("not yet 2s: %v", got)
	}
	now = now.Add(2 * time.Second)
	got := f.Poll()
	if len(got) != 1 || got[0] != path {
		t.Fatalf("got %v", got)
	}
}

func TestPeekNameFromZip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("MyMod/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(`{"Name":"Peeked Mod"}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	inf := Identify(path)
	if inf.Name != "Peeked Mod" || inf.ModID != 0 {
		t.Fatalf("got %+v", inf)
	}
}
