//go:build !windows

package launchsvc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestBisectRunStateMachine(t *testing.T) {
	origSettle, origGrace := bisectSceneSettle, bisectStartupGrace
	bisectSceneSettle, bisectStartupGrace = 0, 0
	t.Cleanup(func() {
		bisectSceneSettle, bisectStartupGrace = origSettle, origGrace
	})

	svc, p := startEnv(t)
	g := game.Find("stardew")
	sl := svc.profileSlot(g, p.ID)
	writeRun := func(id string, exit *launch.Exit) {
		t.Helper()
		modsDir, err := svc.profiles.ModsDir("stardew", p.ID)
		if err != nil {
			t.Fatal(err)
		}
		dir := runsDir(modsDir)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := datadir.WriteJSON(filepath.Join(dir, "index.json"), runIndex{Runs: []Run{{ID: id, Exit: exit}}}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(runLogPath(dir, id), []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	setStatus := func(st State, err string) {
		svc.mu.Lock()
		svc.status[keyOf(sl)] = Status{Game: "stardew", Install: sl.inst, State: st, Error: err, Profile: p.ID}
		svc.mu.Unlock()
	}
	setLines := func(lines ...launch.Entry) {
		buf := &launch.Buffer{}
		for _, e := range lines {
			buf.Add(e)
		}
		svc.mu.Lock()
		svc.logs[keyOf(sl)] = session{buf: buf, profile: p.ID}
		svc.mu.Unlock()
	}
	run := func() *bisectRun {
		return &bisectRun{s: svc, gameID: "stardew", profileID: p.ID, sl: sl}
	}

	t.Run("running crash stops unhealthy", func(t *testing.T) {
		r := run()
		setLines(launch.Entry{Level: launch.Alert, Message: "fatal startup error"})
		got := r.running(t.Context())
		if got == nil || got.err != nil || got.healthy {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("running scene settled stops healthy", func(t *testing.T) {
		r := run()
		r.scenes = &sceneWatch{}
		setLines(launch.Entry{Message: "Bridge plugin alive after scene MainMenu: True"})
		got := r.running(t.Context())
		if got == nil || got.err != nil || !got.healthy {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("failed before started is an error", func(t *testing.T) {
		r := run()
		setStatus(Failed, "no exe")
		got := r.step(t.Context())
		if got == nil || got.err == nil {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("failed after started uses lastRun", func(t *testing.T) {
		writeRun("after", &launch.Exit{Code: 1})
		r := run()
		r.started = true
		r.beforeRunID = "before"
		setStatus(Failed, "exited")
		got := r.step(t.Context())
		if got == nil || got.err != nil || got.healthy || got.summary.Crashed != true {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("idle with a new run id maps summaryHealthy", func(t *testing.T) {
		writeRun("fresh", nil)
		r := run()
		r.beforeRunID = "old"
		setStatus(Idle, "")
		got := r.step(t.Context())
		if got == nil || got.err != nil || !got.healthy {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("lastRun with the same run id returns no verdict", func(t *testing.T) {
		writeRun("same", nil)
		r := run()
		r.beforeRunID = "same"
		got := r.lastRun()
		if got == nil || got.healthy || got.summary.Crashed {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("stopWith healthy", func(t *testing.T) {
		r := run()
		got := r.stopWith(t.Context(), true)
		if got == nil || got.err != nil || !got.healthy {
			t.Fatalf("got %+v", got)
		}
	})
}

func TestStopBisectRunStopsOnlyWhenRunning(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	sl := svc.profileSlot(g, p.ID)
	if err := svc.stopBisectRun(t.Context(), sl); err != nil {
		t.Fatal(err)
	}
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	fakeProc(t, svc, strconv.Itoa(cmd.Process.Pid), "", "/g/StardewModdingAPI", "--mods-path", mods)
	svc.mu.Lock()
	svc.status[keyOf(sl)] = Status{Game: "stardew", Install: sl.inst, State: Running, Profile: p.ID}
	svc.mu.Unlock()
	if err := svc.stopBisectRun(context.Background(), sl); err != nil {
		t.Fatal(err)
	}
	if svc.current(sl).State != Idle {
		t.Fatalf("state = %s", svc.current(sl).State)
	}
}

func TestUpdatesBeforePlayHonoursTheProfileOverride(t *testing.T) {
	svc, p := startEnv(t)
	if svc.UpdatesBeforePlay("stardew", p.ID) {
		t.Fatal("default must not update before play")
	}
	if _, err := svc.profiles.SetOverride("stardew", p.ID, "updateModsBeforePlayDefault", "true"); err != nil {
		t.Fatal(err)
	}
	if !svc.UpdatesBeforePlay("stardew", p.ID) {
		t.Fatal("profile override was ignored")
	}
}
