package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// Turning separate saves on copies the shared saves in once, leaves a folder that already holds saves alone, and
// still turns on when the game has no shared saves yet.
func TestSetSeparateSavesCopiesSharedSavesOnce(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	st, err := settings.OpenIn(e.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	e.settings = st
	e.home = t.TempDir()
	shared, err := game.SavesDir(e.home, st.Get(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, shared, "Farm/Farm_123", "shared")
	p := mustCreate(t, e, "Farm")
	if _, err := e.SetSeparateSaves("stardew", p.ID, true, true); err != nil {
		t.Fatal(err)
	}
	own, err := e.SavesFolder("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(own, "Farm", "Farm_123")); got != "shared" {
		t.Fatalf("copied save = %q, want the shared one", got)
	}

	writeFile(t, own, "Farm/Farm_123", "mine")
	writeFile(t, shared, "Farm/Farm_123", "shared again")
	if _, err := e.SetSeparateSaves("stardew", p.ID, true, true); err != nil {
		t.Fatal(err)
	}
	if got := testfs.Read(t, filepath.Join(own, "Farm", "Farm_123")); got != "mine" {
		t.Fatalf("own save = %q, want the one already there", got)
	}

	if err := os.RemoveAll(shared); err != nil {
		t.Fatal(err)
	}
	q := mustCreate(t, e, "Empty")
	if _, err := e.SetSeparateSaves("stardew", q.ID, true, true); err != nil {
		t.Fatalf("turning on with no shared saves: %v", err)
	}
	if !e.SeparateSaves("stardew", q.ID) {
		t.Fatal("the flag must stay on when there was nothing to copy")
	}
	empty, err := e.SavesFolder("stardew", q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ents, err := os.ReadDir(empty); err == nil && len(ents) > 0 {
		t.Fatalf("own saves folder = %v, want nothing copied", ents)
	}
}
