package launchsvc

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestRecordExitWithoutCrashLogMarksCrashed(t *testing.T) {
	svc, p, _, _ := runEnv(t)
	g := game.Find("stardew")
	svc.mu.Lock()
	svc.logs[keyOf(g)] = session{haveExit: true, exit: launch.Exit{Code: 134, Signal: "SIGABRT"}}
	svc.mu.Unlock()
	svc.record(g, p.ID, time.Now(), false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Outcome != launch.OutcomeCrashed {
		t.Fatalf("crashed = %#v, %v", runs, err)
	}
	if runs[0].Exit == nil || runs[0].Exit.Code != 134 || runs[0].Exit.Signal != "SIGABRT" {
		t.Fatalf("exit = %#v", runs[0].Exit)
	}
}

func TestRecordUserStopNotCrashed(t *testing.T) {
	svc, p, _, _ := runEnv(t)
	g := game.Find("stardew")
	svc.mu.Lock()
	svc.logs[keyOf(g)] = session{haveExit: true, exit: launch.Exit{Code: 137, Signal: "SIGKILL", Stopped: true}}
	svc.mu.Unlock()
	svc.record(g, p.ID, time.Now(), false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Outcome != launch.OutcomeRan {
		t.Fatalf("stop = %#v, %v", runs, err)
	}
}

func TestRecordUserStopIsNeverACrashWhateverTheLogSays(t *testing.T) {
	svc, p, cfg, home := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Stardew's runtime often segfaults while it is being stopped, and a frozen game the player stops may have
	// logged a fatal error first.
	writeOwnedLog(t, cfg, home, mods, "[19:43:51 ALERT SMAPI] The game crashed: boom\n")
	g := game.Find("stardew")
	svc.mu.Lock()
	svc.logs[keyOf(g)] = session{haveExit: true, exit: launch.Exit{Code: 139, Signal: "SIGSEGV", Stopped: true}}
	svc.mu.Unlock()
	svc.record(g, p.ID, time.Now(), false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Outcome != launch.OutcomeRan || runs[0].Cause != nil {
		t.Fatalf("stop = %#v, %v", runs, err)
	}
}

func TestRunEndNotificationTextExit(t *testing.T) {
	title, body := RunEndNotificationText("Stardew Valley", launch.Summary{
		Crashed: true,
		Exit:    launch.Exit{Code: 134, Signal: "SIGABRT"},
	})
	if title != "Stardew Valley crashed" || body != "exited with code 134 (SIGABRT)" {
		t.Fatalf("got %q / %q", title, body)
	}
}
