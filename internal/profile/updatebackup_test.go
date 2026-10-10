package profile

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func TestTheSaveBackupBeforeAnUpdateFollowsBackupBeforePlay(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, err := e.Create(folderGame, "S")
	if err != nil {
		t.Fatal(err)
	}
	own, err := e.SavesFolder(folderGame, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, own, "Slot_00000001.save", "the profile's save")
	target, err := backup.TargetFor(e.base, settings.Defaults(), folderGame, nil)
	if err != nil {
		t.Fatal(err)
	}
	count := func() int {
		t.Helper()
		zips, err := backup.List(target.Dir, gamepkg.SaveLayout(folderGame, own))
		if err != nil {
			t.Fatal(err)
		}
		return len(zips)
	}
	if err := e.saveBackup(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 1 {
		t.Fatalf("backups after an update = %d, want 1 holding the profile's own saves", n)
	}
	if _, err := e.SetOverride(folderGame, p.ID, "backupBeforePlay", settings.BackupBeforePlayNever); err != nil {
		t.Fatal(err)
	}
	writeFile(t, own, "Slot_00000002.save", "a later save")
	if err := e.saveBackup(folderGame, p.ID); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 1 {
		t.Fatalf("backups with backupBeforePlay never = %d, want still 1", n)
	}
}
