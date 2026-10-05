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
