package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRepairFolderRecordsFollowsARenamedFolder(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	e.item(t, "local-a", map[string]string{
		"[FS]Kyuya‘s hats Pack/manifest.json": manifestJSON("X.Hats"),
		"Plain/manifest.json":                 manifestJSON("X.Plain"),
	})
	p := mustCreate(t, e, "P")
	if _, err := e.AddEntry("stardew", p.ID, "local-a", Source{Kind: KindLocal, Name: "a.zip"}); err != nil {
		t.Fatal(err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	cur, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	const stale = "[FS]Kyuya��s hats Pack"
	for i, c := range cur.Entries[0].Mods {
		if c.ID == "smapi:X.Hats" {
			cur.Entries[0].Mods[i].Folder = stale
		}
	}
	if err := writeProfile(dir, cur); err != nil {
		t.Fatal(err)
	}

	n, err := e.RepairFolderRecords()
	if err != nil || n != 1 {
		t.Fatalf("repaired %d, %v; want 1", n, err)
	}
	got, err := e.read("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got.Entries[0].Mods {
		if c.ID == "smapi:X.Hats" && c.Folder != "[FS]Kyuya‘s hats Pack" {
			t.Fatalf("folder = %q", c.Folder)
		}
	}
	if _, err := e.SetModEnabled("stardew", p.ID, got.Entries[0].Key, "smapi:X.Hats", false); err != nil {
		t.Fatalf("switching the mod off after the repair: %v", err)
	}
	if n, err := e.RepairFolderRecords(); err != nil || n != 0 {
		t.Fatalf("second pass repaired %d, %v", n, err)
	}
}

func TestRepairFolderRecordsLeavesAmbiguousAndMissingFoldersAlone(t *testing.T) {
	t.Parallel()
	mods := t.TempDir()
	for _, d := range []string{"k/A‘B", "k/A’B"} {
		if err := os.MkdirAll(filepath.Join(mods, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	p := Profile{Entries: []Entry{{Key: "k", Mods: []Component{{Folder: "A�B"}, {Folder: "Z�B"}}}}}
	if n := repairFolderRecords(&p, mods); n != 0 {
		t.Fatalf("repaired %d, want 0: two folders fit the first, none the second", n)
	}
}
