package profile

import (
	"path/filepath"
	"testing"
)

func TestOverwriteFilesCountAndClear(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p := mustCreate(t, e, "Lobby")
	if n, err := e.OverwriteFiles("stardew", p.ID); err != nil || n != 0 {
		t.Fatalf("no overwrite folder: %d, %v", n, err)
	}
	dir, err := e.profileDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, OverwriteDir), "profile/BepInEx/LogOutput.log", "log")
	writeFile(t, filepath.Join(dir, OverwriteDir), "profile/new.cfg", "cfg")
	if n, err := e.OverwriteFiles("stardew", p.ID); err != nil || n != 2 {
		t.Fatalf("two files: %d, %v", n, err)
	}
	if err := e.ClearOverwrite("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := e.OverwriteFiles("stardew", p.ID); err != nil || n != 0 {
		t.Fatalf("after clearing: %d, %v", n, err)
	}
	e.Running = func(string, string) bool { return true }
	if err := e.ClearOverwrite("stardew", p.ID); err == nil {
		t.Fatal("a running game's overwrite folder was cleared")
	}
}
