package profile

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

const godotGame = "dome-keeper"

func nexusSource(modID, fileID int) Source {
	return Source{Kind: KindNexus, Name: "Mod", ModID: modID, FileID: fileID}
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	var out []string
	for _, f := range r.File {
		out = append(out, f.Name)
	}
	return out
}

func TestZippedModIsPlacedAsOneZipRecordedAndRemoved(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create(godotGame, "G")
	if err != nil {
		t.Fatal(err)
	}
	archive := zipOf(t, "m.zip", map[string]string{"mods-unpacked/Me-Mod/manifest.json": "{}", "mods-unpacked/Me-Mod/main.gd": "x"})
	res, err := e.InstallSource(t.Context(), godotGame, p.ID, archive, nexusSource(5, 9))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Profile.Entries) != 1 || !res.Profile.Entries[0].Package || res.Profile.Entries[0].Mods[0].ID != "folder:nexus-5" {
		t.Fatalf("entries = %+v", res.Profile.Entries)
	}
	key := res.Profile.Entries[0].Key
	if err := e.SyncPackages(godotGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(godotGame, p.ID)
	placed := filepath.Join(dir, "mods", key+".zip")
	if names := zipNames(t, placed); len(names) != 2 || names[0] != "mods-unpacked/Me-Mod/main.gd" {
		t.Fatalf("zip = %v", names)
	}
	if rec := readPlaced(dir)["mods/"+key+".zip"]; rec.Hash == "" || rec.Key != key {
		t.Fatalf("record = %+v", rec)
	}

	// A copy the player edited stays and is reported; it is held aside when the mod goes.
	if err := safefs.WriteFile(placed, []byte("edited"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(godotGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if b, err := safefs.ReadFile(placed); err != nil || string(b) != "edited" {
		t.Fatalf("an edited copy was overwritten: %q", b)
	}
	if _, err := e.RemoveEntries(godotGame, p.ID, []string{key}); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(godotGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(placed); !os.IsNotExist(err) {
		t.Fatalf("zip left after removal: %v", err)
	}
	if left, _ := os.ReadDir(filepath.Join(dir, zipsDir)); len(left) != 0 {
		t.Fatalf("cached zips left: %v", left)
	}
}

func TestZippedModUpdateReplacesTheOldZip(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(godotGame, "G")
	v1 := zipOf(t, "a.zip", map[string]string{"mods-unpacked/Me-Mod/main.gd": "1"})
	first, err := e.InstallSource(t.Context(), godotGame, p.ID, v1, nexusSource(5, 9))
	if err != nil {
		t.Fatal(err)
	}
	v2 := zipOf(t, "b.zip", map[string]string{"mods-unpacked/Me-Mod/main.gd": "2"})
	res, err := e.InstallSource(t.Context(), godotGame, p.ID, v2, nexusSource(5, 12).WithReplacing(9))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || len(res.Profile.Entries) != 1 {
		t.Fatalf("updated = %v, entries = %v", res.Updated, keysOf(res.Profile))
	}
	if err := e.SyncPackages(godotGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(godotGame, p.ID)
	if _, err := os.Stat(filepath.Join(dir, "mods", first.Profile.Entries[0].Key+".zip")); !os.IsNotExist(err) {
		t.Fatalf("old zip kept: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mods", res.Profile.Entries[0].Key+".zip")); err != nil {
		t.Fatal(err)
	}
}
