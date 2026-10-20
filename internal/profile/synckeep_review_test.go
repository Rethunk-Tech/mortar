package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A profile copy with no placed record that differs from the store (an older install's file) is replaced by the
// package's file, the old bytes are kept under replaced/, and the record names the placed file.
func TestAnUnrecordedDifferingCopyIsReplacedAndKept(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"mc.package": "store bytes"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(folderGame, p.ID)
	copyPath := filepath.Join(dir, "Mods", "mc.package")
	if err := os.WriteFile(copyPath, []byte("stale bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, placedFile)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(copyPath); string(got) != "store bytes" {
		t.Fatalf("the stale copy still blocks the package: %q", got)
	}
	if got, _ := fsx.ReadFile(filepath.Join(dir, replacedDir, "Mods", "mc.package")); string(got) != "stale bytes" {
		t.Fatalf("the replaced bytes were not kept: %q", got)
	}
	if rec, ok := readPlaced(dir)["Mods/mc.package"]; !ok || rec.Size < 0 {
		t.Fatalf("the placed file is not recorded: %+v, %v", rec, ok)
	}

	// Taking the package away and bringing it back must not restore the replaced copy.
	cur, _ := e.Get(folderGame, p.ID)
	if _, err := e.RemoveEntry(folderGame, p.ID, cur.Entries[0].Key); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := fsx.ReadFile(copyPath); string(got) != "store bytes" {
		t.Fatalf("the replaced copy came back: %q", got)
	}
}

func TestReplacedFilesCanBeRestoredAsAChangedCopyOrDeleted(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create(folderGame, "S")
	zip := zipOf(t, "a.zip", map[string]string{"mc.package": "store bytes"})
	if _, err := e.InstallSource(t.Context(), folderGame, p.ID, zip, cfSource(10)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	dir, _ := e.ProfileDir(folderGame, p.ID)
	copyPath := filepath.Join(dir, "Mods", "mc.package")
	if err := os.WriteFile(copyPath, []byte("old bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, placedFile)); err != nil {
		t.Fatal(err)
	}
	if err := e.SyncPackages(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	got, err := e.ReplacedFiles(folderGame, p.ID)
	if err != nil || len(got) != 1 || got[0].Path != "Mods/mc.package" || got[0].Size != int64(len("old bytes")) {
		t.Fatalf("replaced = %+v, %v", got, err)
	}
	if err := e.RestoreReplaced(folderGame, p.ID, "Mods/mc.package"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := e.SyncPackages(folderGame, p.ID); err != nil {
			t.Fatal(err)
		}
		if b, _ := fsx.ReadFile(copyPath); string(b) != "old bytes" {
			t.Fatalf("a restored file was replaced again: %q", b)
		}
	}
	if left, _ := e.ReplacedFiles(folderGame, p.ID); len(left) != 0 {
		t.Fatalf("restore left %+v", left)
	}
	if err := e.DeleteReplaced(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
}
