package storecheck

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func zipFile(t *testing.T, dir, name string, files map[string]string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	f, err := fsx.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for n, body := range files {
		w, _ := zw.Create(n)
		_, _ = w.Write([]byte(body))
	}
	if err := errors.Join(zw.Close(), f.Close()); err != nil {
		t.Fatal(err)
	}
	return p
}

func newService(t *testing.T) (*Service, string, *[]queue.Request) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	downloads := t.TempDir()
	var added []queue.Request
	return New(Deps{
		Items:      items,
		Source:     func(string, string) profile.Source { return profile.Source{Name: "Pack.zip"} },
		Add:        func(r []queue.Request) ([]queue.Item, error) { added = append(added, r...); return nil, nil },
		ArchiveDir: func() string { return downloads },
	}), downloads, &added
}

func TestCheckFindsDamageAndRepairReextractsFromTheArchive(t *testing.T) {
	s, downloads, _ := newService(t)
	archive := zipFile(t, downloads, "Pack.zip", map[string]string{"Mod/a.txt": "aaa"})
	key, err := s.d.Items.AddArchive("stardew", archive)
	if err != nil {
		t.Fatal(err)
	}
	if sum, err := s.Check(context.Background()); err != nil || sum.Checked != 1 || len(sum.Damaged) != 0 {
		t.Fatalf("clean check = %+v, %v", sum, err)
	}
	dir, _ := s.d.Items.Path("stardew", key)
	if err := os.Remove(filepath.Join(dir, "Mod", "a.txt")); err != nil {
		t.Fatal(err)
	}
	sum, err := s.Check(context.Background())
	if err != nil || len(sum.Damaged) != 1 || sum.Damaged[0].Missing != 1 || sum.Damaged[0].Name != "Pack.zip" {
		t.Fatalf("damaged check = %+v, %v", sum, err)
	}
	if got, err := s.Repair("stardew", "p1", key); err != nil || got.Status != "restored" {
		t.Fatalf("repair = %+v, %v", got, err)
	}
	if sum, _ := s.Check(context.Background()); len(sum.Damaged) != 0 {
		t.Fatalf("still damaged after repair: %+v", sum)
	}
}

func TestRepairSaysWhenTheArchiveIsGoneAndKeepsTheItem(t *testing.T) {
	s, downloads, _ := newService(t)
	archive := zipFile(t, downloads, "Pack.zip", map[string]string{"Mod/a.txt": "aaa"})
	key, _ := s.d.Items.AddArchive("stardew", archive)
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Repair("stardew", "p1", key); err == nil {
		t.Fatal("repair without an archive succeeded")
	}
	if _, err := s.d.Items.Path("stardew", key); err != nil {
		t.Fatalf("item lost: %v", err)
	}
}

func TestRepairQueuesANexusFileAndRestoresOnQueueFailure(t *testing.T) {
	s, downloads, added := newService(t)
	key := store.NexusKey(12, 34)
	if err := s.d.Items.AddArchiveKey("stardew", key, zipFile(t, downloads, "n.zip", map[string]string{"Mod/a.txt": "a"})); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Repair("stardew", "p1", key); err != nil || got.Status != "queued" {
		t.Fatalf("repair = %+v, %v", got, err)
	}
	if len(*added) != 1 || (*added)[0].ModID != 12 || (*added)[0].FileID != 34 || (*added)[0].Profile != "p1" {
		t.Fatalf("queued = %+v", *added)
	}
	s2, d2, _ := newService(t)
	s2.d.Add = func([]queue.Request) ([]queue.Item, error) { return nil, errors.New("signed out") }
	if err := s2.d.Items.AddArchiveKey("stardew", key, zipFile(t, d2, "n.zip", map[string]string{"Mod/a.txt": "a"})); err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Repair("stardew", "p1", key); err == nil {
		t.Fatal("queue failure not reported")
	}
	if _, err := s2.d.Items.Path("stardew", key); err != nil {
		t.Fatalf("damaged copy not put back: %v", err)
	}
}

func TestPassWaitsWhileBusyThenVerifies(t *testing.T) {
	s, downloads, _ := newService(t)
	key, _ := s.d.Items.AddArchive("stardew", zipFile(t, downloads, "Pack.zip", map[string]string{"a": "a"}))
	s.d.Busy = func() bool { return true }
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	s.pass(ctx)
	if due, _ := s.d.Items.Refs(true, time.Now()); len(due) != 1 || due[0].Key != key {
		t.Fatalf("busy pass verified: %v", due)
	}
}

func TestCheckBaselinesANexusItemFromItsArchiveWhenTheMD5Matches(t *testing.T) {
	s, downloads, _ := newService(t)
	archive := zipFile(t, downloads, "Pack-5-1-0-1700000000.zip", map[string]string{"Mod/a.txt": "aaa"})
	md5, err := fsx.MD5(archive)
	if err != nil {
		t.Fatal(err)
	}
	key := store.NexusKey(5, 9)
	if err := s.d.Items.AddArchiveKey("stardew", key, archive); err != nil {
		t.Fatal(err)
	}
	dir, _ := s.d.Items.Path("stardew", key)
	if err := os.Remove(filepath.Join(dir, "..", "..", ".manifests", "stardew", key+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Mod", "a.txt"), []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	s.d.NexusMD5 = func(context.Context, string, int, int) (string, error) { return "0000", nil }
	if sum, _ := s.Check(context.Background()); len(sum.Damaged) != 0 {
		t.Fatalf("a different MD5 must fall back to the current files: %+v", sum)
	}
	if err := os.Remove(filepath.Join(dir, "..", "..", ".manifests", "stardew", key+".json")); err != nil {
		t.Fatal(err)
	}
	s.d.NexusMD5 = func(context.Context, string, int, int) (string, error) { return md5, nil }
	sum, err := s.Check(context.Background())
	if err != nil || len(sum.Damaged) != 1 || sum.Damaged[0].Changed != 1 {
		t.Fatalf("baseline from the archive = %+v, %v", sum, err)
	}
}
