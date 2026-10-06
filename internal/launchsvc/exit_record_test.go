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
			stopAt, err := time.ParseInLocation(time.DateTime, time.Now().Format(time.DateOnly)+" "+tc.stopAt, time.Local)
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

func TestCrashedBeforeAcrossMidnight(t *testing.T) {
	local := func(v string) time.Time {
		at, err := time.ParseInLocation(time.DateTime, v, time.Local)
		if err != nil {
			t.Fatal(err)
		}
		return at
	}
	// The session starts at 23:50 on the 4th, crashes at 00:05 on the 5th, and is stopped at 23:55 on the 4th or
	// 00:10 on the 5th; by time of day alone the crash would read as before either stop.
	lines := "[23:50:00 INFO  SMAPI] SMAPI 4.1.10 with Stardew Valley 1.6.15\n" +
		"[00:05:00 ALERT SMAPI] The game crashed: boom\n"
	start := local("2026-10-04 23:50:00").UTC().Format("2006-01-02T15:04:05")
	header := "[23:50:00 TRACE SMAPI] Log started at " + start + " UTC\n" + lines
	logEnd := local("2026-10-05 00:05:00")
	for _, tc := range []struct {
		name, text, stop string
		crashed          bool
	}{
		{"header, stop after the crash", header, "2026-10-05 00:10:00", true},
		{"header, stop before midnight", header, "2026-10-04 23:55:00", false},
		{"file time, stop after the crash", lines, "2026-10-05 00:10:00", true},
		{"file time, stop before midnight", lines, "2026-10-04 23:55:00", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// A wrong log end shows that the header, when present, is what dates the lines.
			end := logEnd
			if tc.text == header {
				end = logEnd.AddDate(0, 0, 3)
			}
			if got := crashedBefore(tc.text, end, local(tc.stop)); got != tc.crashed {
				t.Fatalf("crashedBefore = %v, want %v", got, tc.crashed)
			}
		})
	}
}

func TestRunEndNotificationTextExit(t *testing.T) {
	title, body, _ := RunEndNotificationText("Stardew Valley", launch.Summary{
		Crashed: true,
		Exit:    launch.Exit{Code: 134, Signal: "SIGABRT"},
	})
	if title != "Stardew Valley crashed" || body != "exited with code 134 (SIGABRT)" {
		t.Fatalf("got %q / %q", title, body)
	}
}
