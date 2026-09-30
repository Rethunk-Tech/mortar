//go:build !windows

package launchsvc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/game/stardew"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
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

func TestStartInstallsAMissingLoaderBeforeLaunching(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("LOCALAPPDATA", data)
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "Stardew Valley.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := set.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = folder }); err != nil {
		t.Fatal(err)
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profiles.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), set, profiles)
	asked, release := make(chan string), make(chan struct{})
	svc.EnsureLoader = func(_ context.Context, id string) error {
		asked <- id
		<-release
		return errors.New("boom")
	}
	if err := svc.Start("stardew", p.ID, false); err != nil {
		t.Fatal(err)
	}
	if id := <-asked; id != "stardew" {
		t.Fatalf("ensured %q", id)
	}
	if err := svc.Start("stardew", p.ID, false); err == nil {
		t.Fatal("a second Start while the loader installs must be refused")
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.mu.Lock()
		busy := svc.preparing["stardew"]
		svc.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("still preparing after the install failed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st := svc.current("stardew"); st.State != Idle {
		t.Fatalf("state after a failed install = %v", st.State)
	}
}

func TestSendRunsThroughTheBridgeAndEchoesTheCommand(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profiles.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, profiles)
	if err := svc.Send("stardew", "help"); err == nil {
		t.Fatal("Send while idle must fail")
	}
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address is %T", ln.Addr())
	}
	defer func() { _ = ln.Close() }()
	got := make(chan string, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		r := bufio.NewReader(c)
		token, _ := r.ReadString('\n')
		cmd, _ := r.ReadString('\n')
		got <- token + cmd
		_, _ = c.Write([]byte("ok\n"))
	}()
	modsDir, _ := profiles.ModsDir("stardew", p.ID)
	dir := filepath.Join(modsDir, "bridge-"+(stardew.Game{}).BridgeVersion(), "MortarSmapiBridge")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	state := fmt.Sprintf(`{"port":%d,"token":"tok","pid":%d}`, addr.Port, os.Getpid())
	if err := os.WriteFile(filepath.Join(dir, "mortar-smapi-bridge.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	buf := &launch.Buffer{}
	svc.logs["stardew"] = buf
	svc.status["stardew"] = Status{Game: "stardew", State: Running, Profile: p.ID}
	if err := svc.Send("stardew", "  help  "); err != nil {
		t.Fatal(err)
	}
	if line := <-got; line != "tok\nhelp\n" {
		t.Fatalf("bridge saw %q", line)
	}
	lines := buf.Lines()
	if len(lines) != 1 || lines[0].Mod != "Mortar" || lines[0].Message != "> help" {
		t.Fatalf("console = %+v", lines)
	}
}
