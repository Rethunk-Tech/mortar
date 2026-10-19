package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// A profile moved to a folder on another filesystem (/var/tmp is not the tmpfs the store lives on here) keeps working
// from its usual place, and comes back whole.
func TestProfileMovesToAFolderAndHome(t *testing.T) {
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	home := filepath.Join(s.root, "stardew", p.ID)
	mod := filepath.Join(home, "mods", "x", "file.dll")
	if err := os.MkdirAll(filepath.Dir(mod), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mod, []byte("dll"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()

	if _, err := s.MoveToFolder("stardew", p.ID, "relative"); err == nil {
		t.Error("a relative folder was accepted")
	}
	if _, err := s.MoveToFolder("stardew", p.ID, s.root); err == nil {
		t.Error("a folder under the data folder was accepted")
	}
	moved, err := s.MoveToFolder("stardew", p.ID, parent)
	want := filepath.Join(parent, "stardew-"+p.ID)
	if err != nil || moved.Folder != want {
		t.Fatalf("moved = %+v, %v", moved, err)
	}
	if b, err := os.ReadFile(filepath.Clean(filepath.Join(want, "mods", "x", "file.dll"))); err != nil || string(b) != "dll" {
		t.Fatalf("file in the folder = %q, %v", b, err)
	}
	if _, err := s.SetAppearance("stardew", p.ID, "teal", "", ""); err != nil {
		t.Fatalf("write through the link: %v", err)
	}
	listed, err := s.List("stardew")
	if err != nil || len(listed) != 1 || listed[0].Folder != want || listed[0].Color != "teal" {
		t.Fatalf("list = %+v, %v", listed, err)
	}
	if c, err := s.CheckFolder("stardew", p.ID); err != nil || !c.OK {
		t.Fatalf("check = %+v, %v", c, err)
	}
	if err := os.Rename(want, want+".away"); err != nil {
		t.Fatal(err)
	}
	if c, err := s.CheckFolder("stardew", p.ID); err != nil || c.OK || c.Problem == "" {
		t.Fatalf("a missing folder checked as %+v, %v", c, err)
	}
	if err := os.Rename(want+".away", want); err != nil {
		t.Fatal(err)
	}

	back, err := s.MoveHome("stardew", p.ID)
	if err != nil || back.Folder != "" || back.Color != "teal" {
		t.Fatalf("home = %+v, %v", back, err)
	}
	if fi, err := os.Lstat(home); err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("usual place after the move home: %v, %v", fi, err)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("the folder was left behind: %v", err)
	}

	if _, err := s.MoveToFolder("stardew", p.ID, parent); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(s.trash, "stardew", p.ID, "mods", "x", "file.dll")); err != nil || string(b) != "dll" {
		t.Errorf("trashed profile file = %q, %v", b, err)
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Errorf("deleting left the folder behind: %v", err)
	}
}

func TestMovesRefuseALinkToAFolderMortarDidNotMake(t *testing.T) {
	s := newStore(t)
	p := mustCreate(t, s, "Farm")
	home := filepath.Join(s.root, "stardew", p.ID)
	install := filepath.Join(t.TempDir(), "RealInstall")
	if err := os.MkdirAll(install, 0o700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(install, "game.exe")
	if err := os.WriteFile(keep, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "profile.json"), []byte(`{"id":"`+p.ID+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(home); err != nil {
		t.Fatal(err)
	}
	if err := linkDir(home, install); err != nil {
		t.Fatal(err)
	}
	if err := s.bringHome("stardew", p.ID, home); err == nil {
		t.Error("bringHome followed a link to a folder with the wrong name")
	}
	if err := s.relocate("stardew", p.ID, home, filepath.Join(t.TempDir(), "stardew-"+p.ID)); err == nil {
		t.Error("relocate followed a link to a folder with the wrong name")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("the player's folder was touched: %v", err)
	}
}
