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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func TestRunningFollowsProcessesAndLocksProfile(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, profiles := testenv.Stores(t)
	a := testenv.Profile(t, profiles, "stardew", "A")
	b := testenv.Profile(t, profiles, "stardew", "B")
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

func TestLinesReadsTheProfilesLastLog(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	_, profiles := testenv.Stores(t)
	a := testenv.Profile(t, profiles, "stardew", "A")
	b := testenv.Profile(t, profiles, "stardew", "B")
	svc := NewService(home, nil, profiles)
	if got, err := svc.Lines("stardew", a.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no log yet: %v, %v", got, err)
	}
	dir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	modsA, _ := profiles.ModsDir("stardew", a.ID)
	rel, _ := filepath.Rel(home, modsA)
	log := "[19:43:46 INFO  SMAPI] Mods go here: ~/" + rel + "\r\n[19:43:50 ERROR Mod] boom\n  at X\n"
	if err := os.WriteFile(filepath.Join(dir, "SMAPI-latest.txt"), []byte(log), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Lines("stardew", a.ID)
	if err != nil || len(got) != 3 {
		t.Fatalf("got %v, %v", got, err)
	}
	if got[1].Level != launch.Error || got[1].Mod != "Mod" || !got[2].Cont || got[2].Level != launch.Error || got[0].Seq >= got[1].Seq {
		t.Fatalf("got %+v", got)
	}
	if got, err := svc.Lines("stardew", b.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("profile B has no log: %v, %v", got, err)
	}

	buf := &launch.Buffer{}
	buf.Add(launch.Entry{Message: "session"})
	svc.logs["stardew"] = session{buf: buf, profile: b.ID}
	if got, err := svc.Lines("stardew", b.ID); err != nil || len(got) != 1 {
		t.Fatalf("profile B's session: %v, %v", got, err)
	}
	if got, err := svc.Lines("stardew", a.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("B's session is not A's: %v, %v", got, err)
	}

	buf = &launch.Buffer{}
	buf.Add(launch.Entry{Message: "Started without mods"})
	svc.logs["stardew"] = session{buf: buf, vanilla: true}
	if got, err := svc.Lines("stardew", a.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("vanilla session leaked to a profile: %v, %v", got, err)
	}
	if got, err := svc.Lines("stardew", ""); err != nil || len(got) != 1 || got[0].Message != "Started without mods" {
		t.Fatalf("empty profile id should see the vanilla session: %v, %v", got, err)
	}
}

func TestGameClosingEndsTheConsoleWithAMortarLine(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	_, profiles := testenv.Stores(t)
	a := testenv.Profile(t, profiles, "stardew", "A")
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
	svc.logs["stardew"] = session{buf: buf, profile: a.ID}
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

// startEnv is a Stardew folder without SMAPI and one profile, so Start goes through EnsureLoader.
func startEnv(t *testing.T) (*Service, profile.Profile) {
	t.Helper()
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
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	return NewService(t.TempDir(), set, profiles), p
}

func TestPreviewCommandUsesUnsavedLaunchFields(t *testing.T) {
	svc, p := startEnv(t)
	preview := svc.PreviewCommand("stardew", p.ID, `--foo "two words"`, `gamemoderun "mango hud"`, "ONE=value\nTWO=two")
	if preview.Error != "" {
		t.Fatal(preview.Error)
	}
	if len(preview.Env) != 2 || preview.Env[0] != "ONE=value" || preview.Env[1] != "TWO=two" {
		t.Fatalf("env = %#v", preview.Env)
	}
	if len(preview.Argv) < 7 || preview.Argv[1] != "gamemoderun" || preview.Argv[2] != "mango hud" {
		t.Fatalf("argv = %#v", preview.Argv)
	}
	if preview.Argv[len(preview.Argv)-2] != "--foo" || preview.Argv[len(preview.Argv)-1] != "two words" {
		t.Fatalf("argv = %#v", preview.Argv)
	}
}

func TestStartInstallsAMissingLoaderBeforeLaunching(t *testing.T) {
	svc, p := startEnv(t)
	asked, release := make(chan string), make(chan struct{})
	svc.EnsureLoader = func(_ context.Context, id string, _ bool) error {
		asked <- id
		<-release
		return errors.New("boom")
	}
	if err := svc.Start(context.Background(), "stardew", p.ID, false); err != nil {
		t.Fatal(err)
	}
	if id := <-asked; id != "stardew" {
		t.Fatalf("ensured %q", id)
	}
	if err := svc.Start(context.Background(), "stardew", p.ID, false); err == nil {
		t.Fatal("a second Start while the loader installs must be refused")
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.mu.Lock()
		_, busy := svc.preparing["stardew"]
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
	items, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
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
	// The running profile holds an older bridge than the one this build bundles.
	src := filepath.Join(t.TempDir(), "MortarSmapiBridge")
	if err := os.MkdirAll(src, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"Name":"Bridge","Author":"m","Version":"0.9.0","UniqueID":"Rethunk.MortarSmapiBridge","EntryDll":"B.dll"}`
	if err := os.WriteFile(filepath.Join(src, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := items.AddDir("stardew", "bridge-0.9.0", filepath.Dir(src)); err != nil {
		t.Fatal(err)
	}
	if err := profiles.ApplyBundled("stardew", profile.Bundle{Key: "bridge-0.9.0", Source: profile.Source{Kind: profile.SourceMortar, Name: "Bridge"}}); err != nil {
		t.Fatal(err)
	}
	modsDir, _ := profiles.ModsDir("stardew", p.ID)
	dir := filepath.Join(modsDir, "bridge-0.9.0", "MortarSmapiBridge")
	state := fmt.Sprintf(`{"port":%d,"token":"tok","pid":%d}`, addr.Port, os.Getpid())
	if err := os.WriteFile(filepath.Join(dir, "mortar-smapi-bridge.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	buf := &launch.Buffer{}
	svc.logs["stardew"] = session{buf: buf, profile: p.ID}
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

func TestConcurrentStartsLaunchOnce(t *testing.T) {
	svc, p := startEnv(t)
	var ensured atomic.Int32
	release := make(chan struct{})
	svc.EnsureLoader = func(context.Context, string, bool) error {
		ensured.Add(1)
		<-release
		return errors.New("stop here")
	}
	var wg sync.WaitGroup
	var refused atomic.Int32
	gate := make(chan struct{})
	for range 2 {
		wg.Go(func() {
			<-gate
			if svc.Start(context.Background(), "stardew", p.ID, false) != nil {
				refused.Add(1)
			}
		})
	}
	close(gate)
	wg.Wait()
	close(release)
	if refused.Load() != 1 {
		t.Fatalf("%d of two concurrent Starts refused, want 1", refused.Load())
	}
}

func TestStartWithAnInstalledLoaderWaitsForEnsureLoader(t *testing.T) {
	svc, p := startEnv(t)
	folder := svc.settings.Get().GameFolders["stardew"]
	if err := os.WriteFile(filepath.Join(folder, "StardewValley-original"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "StardewValley"), []byte("exec StardewModdingAPI"), 0o600); err != nil {
		t.Fatal(err)
	}
	asked, release := make(chan struct{}), make(chan struct{})
	svc.EnsureLoader = func(context.Context, string, bool) error {
		close(asked)
		<-release
		return errors.New("update failed")
	}
	if err := svc.Start(context.Background(), "stardew", p.ID, false); err != nil {
		t.Fatal(err)
	}
	<-asked
	if st := svc.current("stardew"); st.State != Idle {
		t.Fatalf("launched while the loader was being updated: %v", st.State)
	}
	close(release)
	deadline := time.Now().Add(5 * time.Second)
	for {
		svc.mu.Lock()
		_, busy := svc.preparing["stardew"]
		svc.mu.Unlock()
		if !busy {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("still preparing")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st := svc.current("stardew"); st.State != Idle {
		t.Fatalf("state after a failed update = %v", st.State)
	}
}

func TestAProfileReadiedForLaunchIsRunning(t *testing.T) {
	svc, p := startEnv(t)
	other, err := svc.profiles.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	svc.procDir = t.TempDir()
	svc.preparing["stardew"] = p.ID
	if !svc.Running("stardew", p.ID) {
		t.Fatal("the profile is locked while its loader is still being checked")
	}
	if svc.Running("stardew", other.ID) {
		t.Fatal("only the readied profile is running")
	}
}

func TestAProcessWithoutACommandLineLocksEveryProfileUnlessMortarLaunchedIt(t *testing.T) {
	bare := launch.Process{PID: 1}
	if !credited(bare, "/a", "a", "", false) || !credited(bare, "/b", "b", "", false) {
		t.Fatal("a game Mortar did not start may run any profile")
	}
	if !credited(bare, "/a", "a", "a", false) || credited(bare, "/b", "b", "a", false) {
		t.Fatal("a game Mortar launched runs its launched profile only")
	}
	withArgs := launch.Process{PID: 2, Args: []string{"StardewModdingAPI", "--mods-path", "/a"}}
	if !credited(withArgs, "/a", "a", "", false) || credited(withArgs, "/b", "b", "", false) {
		t.Fatal("a command line names its profile")
	}
	if credited(bare, "/a", "a", "", true) || credited(withArgs, "/a", "a", "", true) {
		t.Fatal("a vanilla launch locks no profile")
	}
}

func TestStartLoaderUsesAppLifetime(t *testing.T) {
	svc, p := startEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	SetLife(svc, ctx)
	saw := make(chan context.Context, 1)
	svc.EnsureLoader = func(ctx context.Context, _ string, _ bool) error {
		saw <- ctx
		return ctx.Err()
	}
	if err := svc.Start(context.Background(), "stardew", p.ID, true); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-saw:
		if got.Err() == nil {
			t.Fatal("EnsureLoader ran with a live context")
		}
	case <-time.After(time.Second):
		t.Fatal("EnsureLoader was not called")
	}
}

const fakeGameEnv = "MORTAR_TEST_FAKE_GAME"

func TestMain(m *testing.M) {
	if os.Getenv(fakeGameEnv) != "" {
		time.Sleep(3 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestStartedGameIsNotCancelledWhenStartReturns(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	svc, p := startEnv(t)
	folder := svc.settings.Get().GameFolders["stardew"]
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// The test binary stands in for the game: with fakeGameEnv set it sleeps, then exits cleanly.
	t.Setenv(fakeGameEnv, "1")
	for _, name := range []string{"StardewValley", "StardewValley-original", "StardewModdingAPI"} {
		if err := os.Symlink(self, filepath.Join(folder, name)); err != nil {
			t.Fatal(err)
		}
	}
	svc.EnsureLoader = func(context.Context, string, bool) error { return nil }
	if err := svc.Start(context.Background(), "stardew", p.ID, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		runs, err := svc.Runs("stardew", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(runs) > 0 {
			if runs[0].DurationMs < 2000 {
				text, _ := svc.RunLog("stardew", p.ID, runs[0].ID)
				t.Fatalf("the game was stopped after %d ms: %q", runs[0].DurationMs, text)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the run was never recorded")
}

func TestStartPresetRefusesAnUnknownPreset(t *testing.T) {
	svc, p := startEnv(t)
	if err := svc.StartPreset(context.Background(), "stardew", p.ID, "nope", false); err == nil {
		t.Fatal("unknown preset accepted")
	}
	svc.mu.Lock()
	_, busy := svc.preparing["stardew"]
	svc.mu.Unlock()
	if busy {
		t.Fatal("a refused launch left the game marked busy")
	}
}
