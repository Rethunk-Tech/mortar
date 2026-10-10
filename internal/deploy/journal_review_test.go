package deploy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
)

// The architecture contract says Purge removes a destination Mortar placed (Done) even when the content changed; a
// game that rewrites a placed file must not leave it behind in the player's folder.
func TestPurgeRemovesAPlacedFileTheGameRewrote(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	ini := filepath.Join(r.install, "doorstop_config.ini")
	write(t, ini, "rewritten by the game")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(ini); err == nil {
		t.Fatal("a file Mortar placed was left behind because the game changed it")
	}
}

// A crash between two checkpoints leaves the first record on disk: no Done flags and no Created list, while the
// files and the folders made for them exist. Recovery must remove the folders Mortar made.
func TestRecoverAfterACrashBetweenCheckpointsRemovesTheFoldersApplyMade(t *testing.T) {
	r := newRig(t)
	r.files = []launchplan.PlanFile{{Src: filepath.Join(r.store, "doorstop_config.ini"), Dst: "a/b/c.ini"}}
	m := r.apply()
	for i := range m.Ops {
		m.Ops[i].Done = false
	}
	m.Created = nil
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(filepath.Join(r.view.JournalDir, journalFile), b, 0o600); err != nil {
		t.Fatal(err)
	}
	d, _ := Get(copyID)
	if err := d.Recover(t.Context(), r.view.JournalDir, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(r.install, "a")); err == nil {
		t.Fatal("recovery left the folder Apply made in the player's tree")
	}
}

// CopyFile writes through DST.mortar-tmp; a crash mid-copy leaves it, and recovery must not leave a Mortar file behind.
func TestRecoverRemovesATempFileACrashedCopyLeft(t *testing.T) {
	r := newRig(t)
	m := r.apply()
	tmp := filepath.Join(r.install, "doorstop_config.ini.mortar-tmp")
	write(t, tmp, "partial")
	d, _ := Get(copyID)
	if err := d.Purge(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(tmp); err == nil {
		t.Fatal("a partial copy of a placed file was left in the install")
	}
}
