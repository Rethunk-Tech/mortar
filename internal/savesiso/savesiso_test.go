package savesiso

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func write(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(p))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSwapKeepsSharedSavesAndProfileSavesApart(t *testing.T) {
	for _, noLink := range []bool{false, true} {
		root := t.TempDir()
		saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
		write(t, filepath.Join(saves, "shared.sav"), "shared")
		write(t, filepath.Join(prof, "mine.sav"), "mine")
		m, err := Apply(journal, saves, prof, noLink)
		if err != nil {
			t.Fatal(err)
		}
		if read(t, filepath.Join(saves, "mine.sav")) != "mine" || exists(filepath.Join(saves, "shared.sav")) {
			t.Fatalf("noLink=%v: the game must see only the profile's saves", noLink)
		}
		write(t, filepath.Join(saves, "new.sav"), "played")
		if err := Purge(m); err != nil {
			t.Fatal(err)
		}
		if read(t, filepath.Join(saves, "shared.sav")) != "shared" || exists(filepath.Join(saves, "mine.sav")) {
			t.Fatalf("noLink=%v: the shared saves must return untouched", noLink)
		}
		if read(t, filepath.Join(prof, "new.sav")) != "played" || read(t, filepath.Join(prof, "mine.sav")) != "mine" {
			t.Fatalf("noLink=%v: the profile keeps what was played", noLink)
		}
		if exists(journal) || exists(saves+heldSuffix) {
			t.Fatalf("noLink=%v: nothing is left behind", noLink)
		}
	}
}

func TestRecoverAfterACrashRestoresSharedSaves(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "Saves"), filepath.Join(root, "p"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "shared.sav"), "shared")
	if _, err := Apply(journal, saves, prof, false); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(journal, saves, prof, false); err == nil {
		t.Fatal("a second swap over an unrecovered one must refuse")
	}
	if err := Recover(journal, func() bool { return true }); err != nil || !exists(journalPath(journal)) {
		t.Fatalf("a running game must be left alone: %v", err)
	}
	if err := Recover(journal, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(saves, "shared.sav")) != "shared" || isLink(saves) {
		t.Fatal("recover must put the shared saves back")
	}
}

func TestNoSharedSavesLeavesNothingBehind(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "Saves"), filepath.Join(root, "p"), filepath.Join(root, "j")
	m, err := Apply(journal, saves, prof, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := Purge(m); err != nil || exists(saves) {
		t.Fatalf("no shared folder means none after: %v", err)
	}
}

func TestPurgeRunAgainKeepsTheWrittenBackSaves(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only folder does not stop a delete on Windows")
	}
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "saves"), filepath.Join(root, "profile"), filepath.Join(root, "j")
	write(t, filepath.Join(prof, "a.sav"), "slot a")
	write(t, filepath.Join(prof, "locked", "b.sav"), "slot b")
	m, err := Apply(journal, saves, prof, true)
	if err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(saves, "locked")
	// A read-only folder stands in for a save file held open on Windows.
	if err := fsx.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := Purge(m); err == nil {
		t.Fatal("the working copy was removed from a read-only folder")
	}
	if err := fsx.Chmod(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Recover(journal, nil); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(prof, "a.sav")) != "slot a" || read(t, filepath.Join(prof, "locked", "b.sav")) != "slot b" || exists(saves) {
		t.Fatal("the second purge lost saves or left the working copy")
	}
}

func TestApplyNeverOverwritesSavesAlreadySetAside(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "shared.sav"), "shared")
	write(t, filepath.Join(saves+heldSuffix, "older.sav"), "older")
	if _, err := Apply(journal, saves, prof, false); err == nil {
		t.Fatal("a second set of shared saves was set aside over the first")
	}
	if read(t, filepath.Join(saves, "shared.sav")) != "shared" || read(t, filepath.Join(saves+heldSuffix, "older.sav")) != "older" || exists(journal) {
		t.Fatal("a refused apply changed something")
	}
}

func TestPurgeStopsWhenSomethingIsWhereTheSharedSavesGoBack(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "shared.sav"), "shared")
	m, err := Apply(journal, saves, prof, false)
	if err != nil {
		t.Fatal(err)
	}
	if !isLink(saves) {
		t.Skip("this system made a copy, not a link")
	}
	// The link went and the game, or the player, made a real folder in its place.
	if err := os.Remove(saves); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(saves, "stray.sav"), "stray")
	if err := Purge(m); err == nil {
		t.Fatal("purge reported success with the shared saves still set aside")
	}
	if read(t, filepath.Join(m.Held, "shared.sav")) != "shared" || read(t, filepath.Join(saves, "stray.sav")) != "stray" || !exists(journalPath(journal)) {
		t.Fatal("purge lost the shared saves, the stray folder or its journal")
	}
}

func TestPurgeOfAHalfCopiedSwapLeavesTheProfileAlone(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "shared.sav"), "shared")
	write(t, filepath.Join(prof, "mine.sav"), "mine")
	m, err := Apply(journal, saves, prof, true)
	if err != nil {
		t.Fatal(err)
	}
	// The crash came before the copy was complete: what is at saves is partial, never to be written back.
	m.Live = false
	if err := os.Remove(filepath.Join(saves, "mine.sav")); err != nil {
		t.Fatal(err)
	}
	if err := Purge(m); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(prof, "mine.sav")) != "mine" || read(t, filepath.Join(saves, "shared.sav")) != "shared" {
		t.Fatal("a partial copy replaced the profile's saves")
	}
}

func TestRecoverWaitsForTheGameAndRefusesADamagedJournal(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "shared.sav"), "shared")
	if _, err := Apply(journal, saves, prof, false); err != nil {
		t.Fatal(err)
	}
	if err := Recover(journal, func() bool { return true }); err != nil || !exists(saves+heldSuffix) || !exists(journalPath(journal)) {
		t.Fatalf("recover undid a running game's swap: %v", err)
	}
	write(t, journalPath(journal), "{not json")
	if err := Recover(journal, nil); err == nil || !exists(saves+heldSuffix) {
		t.Fatalf("a damaged journal was purged blind: %v", err)
	}
	if _, err := Apply(journal, saves, prof, false); err == nil {
		t.Fatal("apply ran over an unrecovered journal")
	}
}

func TestPurgeWithAStaleManifestDoesNothing(t *testing.T) {
	root := t.TempDir()
	saves, prof, journal := filepath.Join(root, "g", "Saves"), filepath.Join(root, "p", "saves"), filepath.Join(root, "j")
	write(t, filepath.Join(saves, "Shared_1"), "shared")
	write(t, filepath.Join(prof, "Prof_1"), "mine")
	m, err := Apply(journal, saves, prof, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := Recover(journal, func() bool { return false }); err != nil {
		t.Fatal(err)
	}
	if err := Purge(m); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(saves, "Shared_1")) != "shared" || read(t, filepath.Join(prof, "Prof_1")) != "mine" {
		t.Fatal("a manifest whose journal is gone replayed over the finished swap")
	}
}
