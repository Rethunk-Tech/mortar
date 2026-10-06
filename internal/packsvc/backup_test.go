package packsvc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func dataHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
}

func TestRestoreOnAnotherComputerQueuesTheModsFromTheirSources(t *testing.T) {
	dataHome(t)
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	for key, id := range map[string]string{"nexus-12-34": "X.A", "local-b": "X.B"} {
		src := t.TempDir()
		if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(`{"Name":"`+id+`","Version":"1.0.0","UniqueID":"`+id+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := items.AddDir("stardew", key, src); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := profiles.AddEntry("stardew", p.ID, "nexus-12-34", profile.Source{Kind: profile.KindNexus, Name: "a.zip", ModID: 12, FileID: 34}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.AddEntry("stardew", p.ID, "local-b", profile.Source{Kind: profile.KindLocal, Name: "b.zip"}); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SetSeparateSaves("stardew", p.ID, true, false); err != nil {
		t.Fatal(err)
	}
	saves, err := profiles.SavesFolder("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(saves, "Farm_1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(saves, "Farm_1", "Farm_1"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "farm.zip")
	size, err := (&Service{Profiles: profiles}).BackupSize("stardew", p.ID)
	if err != nil || size < int64(len("save")) {
		t.Fatalf("size %d, %v", size, err)
	}
	if err := (&Service{Profiles: profiles}).Backup("stardew", p.ID, dest); err != nil {
		t.Fatal(err)
	}

	dataHome(t)
	freshItems, fresh := testenv.Stores(t)
	q := &fakeQueue{}
	s := &Service{Profiles: fresh, Queue: q}
	if _, err := s.Restore(context.Background(), dest, "lethal-company"); err == nil {
		t.Fatal("a backup of another game's profile must be refused")
	}
	res, err := s.Restore(context.Background(), dest, "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Queued != 1 || len(q.got) != 1 || q.got[0].ModID != 12 || q.got[0].FileID != 34 || q.got[0].Profile != res.Profile {
		t.Fatalf("queued %+v (result %+v)", q.got, res)
	}
	// The archive-installed mod came back from the backup itself, so nothing is left to install by hand.
	if len(res.Unavailable) != 0 {
		t.Fatalf("unavailable %v", res.Unavailable)
	}
	all, err := fresh.List("stardew")
	if err != nil || len(all) != 1 || len(all[0].Entries) != 2 || !all[0].SeparateSaves {
		t.Fatalf("restored profiles %+v, %v", all, err)
	}
	dir, err := freshItems.Path("stardew", "local-b")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(dir, "manifest.json")); err != nil || !strings.Contains(string(b), "X.B") {
		t.Fatalf("restored store item: %q, %v", b, err)
	}
	modsDir, err := fresh.ProfileDir("stardew", res.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(modsDir, "mods", "local-b", "manifest.json")); err != nil {
		t.Fatalf("the restored mod was not placed: %v", err)
	}
	restored, err := fresh.SavesFolder("stardew", res.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := fsx.ReadFile(filepath.Join(restored, "Farm_1", "Farm_1")); err != nil || string(b) != "save" {
		t.Fatalf("restored save: %q, %v", b, err)
	}
}
