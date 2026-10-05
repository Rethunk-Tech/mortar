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

func TestRecordUserStopIsACrashOnlyWhenTheLogCrashedFirst(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stopAt  string
		outcome launch.Outcome
	}{
		// A frozen game the player stops after it logged a crash.
		{"crash then stop", "19:44:00", launch.OutcomeCrashed},
		// Stardew's runtime often logs or segfaults while it is being stopped.
		{"stop then crash", "19:43:00", launch.OutcomeRan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, p, cfg, home := runEnv(t)
			mods, err := svc.profiles.ModsDir("stardew", p.ID)
			if err != nil {
				t.Fatal(err)
			}
			writeOwnedLog(t, cfg, home, mods, "[19:43:51 ALERT SMAPI] The game crashed: boom\n")
			stopAt, err := time.ParseInLocation("15:04:05", tc.stopAt, time.Local)
			if err != nil {
				t.Fatal(err)
			}
			g := game.Find("stardew")
			svc.mu.Lock()
			svc.logs[keyOf(g)] = session{haveExit: true, exit: launch.Exit{Code: 139, Signal: "SIGSEGV", Stopped: true}, stoppedAt: stopAt}
			svc.mu.Unlock()
			svc.record(g, p.ID, time.Now(), false)
			runs, err := svc.Runs("stardew", p.ID)
			if err != nil || len(runs) != 1 || runs[0].Outcome != tc.outcome {
				t.Fatalf("stop = %#v, %v", runs, err)
			}
		})
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
