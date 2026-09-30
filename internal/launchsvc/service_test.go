//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func TestRunningFollowsProcessesAndLocksProfile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	a, err := profiles.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	b, err := profiles.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	modsA, _ := profiles.ModsDir("stardew", a.ID)

	svc := NewService(t.TempDir(), nil, profiles)
	svc.procDir = t.TempDir()
	profiles.Running = svc.Running
	if svc.Running("stardew", a.ID) {
		t.Fatal("nothing runs yet")
	}
	pid := filepath.Join(svc.procDir, "42")
	if err := os.MkdirAll(pid, 0o700); err != nil {
		t.Fatal(err)
	}
	cmdline := "/g/StardewModdingAPI\x00--mods-path\x00" + modsA + "\x00"
	if err := os.WriteFile(filepath.Join(pid, "cmdline"), []byte(cmdline), 0o600); err != nil {
		t.Fatal(err)
	}
	if !svc.Running("stardew", a.ID) || svc.Running("stardew", b.ID) {
		t.Fatal("only profile A is running")
	}
	if _, err := profiles.RemoveEntry("stardew", a.ID, "x"); err == nil || err.Error() != "Stardew Valley is running this profile: stop the game first" {
		t.Fatalf("err = %v", err)
	}
	if _, err := profiles.RemoveEntry("stardew", b.ID, "x"); err == nil || err.Error() == "Stardew Valley is running this profile: stop the game first" {
		t.Fatalf("profile B must not be locked: %v", err)
	}
}
