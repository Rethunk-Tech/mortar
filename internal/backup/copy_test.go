package backup

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/saves"
)

func TestRestoreCopyPutsTheGamesOwnCopyBack(t *testing.T) {
	dir, backups := t.TempDir(), t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		testfs.WriteAt(t, p, []byte(body))
	}
	read := func(name string) string {
		b, err := fsx.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	write("worlds_local/W.fwl", "now")
	write("worlds_local/W.db", "now db")
	write("worlds_local/W.db-wal", "now wal")
	write("worlds_local/W_backup_auto-1.fwl", "then")
	write("worlds_local/W_backup_auto-1.db", "then db")
	write("Slot_00000002.save", "slot now")
	write("Slot_00000002.save.ver0", "slot then")
	l := saves.Layout{
		Dir:        dir,
		Files:      []string{"worlds_local/*.fwl", "Slot_*.save", "!*_backup_*"},
		Companions: []string{".db", ".db-wal"},
		Copies:     []string{"{stem}_backup_auto-*{ext}", "{name}.ver?"},
	}
	if got := l.CopiesOf("worlds_local/W.fwl"); len(got) != 1 || got[0].Name != "worlds_local/W_backup_auto-1.fwl" {
		t.Fatalf("copies = %+v", got)
	}
	now := time.Now()
	if err := RestoreCopy(t.Context(), l, backups, "worlds_local/W.fwl", "worlds_local/W_backup_auto-1.fwl", 5, now); err != nil {
		t.Fatal(err)
	}
	if read("worlds_local/W.fwl") != "then" || read("worlds_local/W.db") != "then db" || read("worlds_local/W_backup_auto-1.fwl") != "then" ||
		fsx.IsFile(filepath.Join(dir, "worlds_local", "W.db-wal")) {
		t.Fatal("world not restored from its copy, the copy was moved, or the current write-ahead log was left")
	}
	if err := RestoreCopy(t.Context(), l, backups, "Slot_00000002.save", "Slot_00000002.save.ver0", 5, now.Add(time.Second)); err != nil || read("Slot_00000002.save") != "slot then" {
		t.Fatalf("slot restore: %v", err)
	}
	if err := RestoreCopy(t.Context(), l, backups, "Slot_00000002.save", "../elsewhere", 5, now); err == nil {
		t.Fatal("a path that is not one of the save's copies was restored")
	}
	// A copy the game keeps in a folder beside the saves folder (Vintage Story's /genbackup).
	world := saves.Layout{
		Dir:         filepath.Join(dir, "Saves"),
		Files:       []string{"*.vcdbs"},
		Companions:  []string{".vcdbs-wal"},
		Copies:      []string{"{stem}-[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]_*{ext}"},
		CopyFolders: []string{"Backups"},
	}
	write("Saves/default.vcdbs", "world now")
	write("Saves/default-two.vcdbs", "other world")
	write("Backups/default-2026-10-16_05-26-01.vcdbs", "world then")
	write("Backups/default-two-2026-10-16_05-26-01.vcdbs", "other then")
	if got := world.CopiesOf("default.vcdbs"); len(got) != 1 || got[0].Name != "../Backups/default-2026-10-16_05-26-01.vcdbs" {
		t.Fatalf("copies beside the saves folder = %+v", got)
	}
	if err := RestoreCopy(t.Context(), world, t.TempDir(), "default.vcdbs", "../Backups/default-2026-10-16_05-26-01.vcdbs", 5, now); err != nil || read("Saves/default.vcdbs") != "world then" {
		t.Fatalf("restore from a folder beside the saves folder: %v", err)
	}
	if listed, _ := List(backups, l); len(listed) != 2 {
		t.Fatalf("backups before each restore = %d, want 2", len(listed))
	}
}
