//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
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

func TestLinesReadsLastSessionsLog(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	svc := NewService(t.TempDir(), nil, nil)
	if got, err := svc.Lines("stardew"); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no log yet: %v, %v", got, err)
	}
	dir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	log := "[19:43:46 INFO  SMAPI] hello\r\n[19:43:50 ERROR Mod] boom\n  at X\n"
	if err := os.WriteFile(filepath.Join(dir, "SMAPI-latest.txt"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Lines("stardew")
	if err != nil || len(got) != 3 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[1].Level != launch.Error || got[1].Mod != "Mod" || !got[2].Cont || got[2].Level != launch.Error || got[0].Seq >= got[1].Seq {
		t.Fatalf("got %+v", got)
	}
}

func TestGameClosingEndsTheConsoleWithAMortarLine(t *testing.T) {
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
	mods, _ := profiles.ModsDir("stardew", a.ID)
	svc := NewService(t.TempDir(), nil, profiles)
	svc.procDir = t.TempDir()
	pid := filepath.Join(svc.procDir, "42")
	if err := os.MkdirAll(pid, 0o700); err != nil {
		t.Fatal(err)
	}
	cmdline := "/g/StardewModdingAPI\x00--mods-path\x00" + mods + "\x00"
	if err := os.WriteFile(filepath.Join(pid, "cmdline"), []byte(cmdline), 0o600); err != nil {
		t.Fatal(err)
	}
	g := game.Find("stardew")
	if !svc.poll(g) || svc.current("stardew").State != Running {
		t.Fatal("the running process should mark the game running")
	}
	buf := &launch.Buffer{}
	svc.logs["stardew"] = buf
	if err := os.RemoveAll(pid); err != nil {
		t.Fatal(err)
	}
	if svc.poll(g) || svc.current("stardew").State != Idle {
		t.Fatal("the game should be idle once its process is gone")
	}
	lines := buf.Lines()
	if len(lines) != 1 || lines[0].Mod != "Mortar" || !strings.HasPrefix(lines[0].Message, "Stardew Valley closed") {
		t.Fatalf("lines = %+v", lines)
	}
}
