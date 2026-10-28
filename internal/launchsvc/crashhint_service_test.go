//go:build !windows

package launchsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// CrashHint prefers the run's recorded cause, then a loader log finding that names an enabled mod, then the first
// finding's message; a latest run that did not crash has none.
func TestCrashHintPrecedence(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	_, ps := testenv.Stores(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	zip := tsPackage(t, "Fixture", map[string]string{"plugins/Mod.dll": "x"})
	res, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	modKey := res.Profile.Entries[0].Key
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	modsDir, err := ps.ModsDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, ps)

	setRuns := func(runs ...Run) {
		t.Helper()
		if err := os.MkdirAll(runsDir(modsDir), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := datadir.WriteJSON(filepath.Join(runsDir(modsDir), "index.json"), runIndex{Runs: runs}); err != nil {
			t.Fatal(err)
		}
	}
	setLog := func(body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, "BepInEx"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "BepInEx", "LogOutput.log"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const header = "[Info   :   BepInEx] BepInEx 5.4.22 - Lethal Company\n"
	blamesFixture := header + "[Error  :   BepInEx] Could not load [Fixture 1.0.0] because it has missing dependencies: Lib\n"
	blamesStranger := header + "[Error  :   BepInEx] Could not load [Stranger 1.0.0] because it has missing dependencies: Alpha\n"
	crashed := Run{ID: "r1", Outcome: launch.OutcomeCrashed}

	setRuns()
	if h, err := svc.CrashHint(lc, p.ID); h != nil || err != nil {
		t.Fatalf("no runs: %+v %v", h, err)
	}

	setLog(blamesFixture)
	setRuns(Run{ID: "r2", Outcome: launch.OutcomeRan}, crashed)
	if h, err := svc.CrashHint(lc, p.ID); h != nil || err != nil {
		t.Fatalf("latest run ran: %+v %v", h, err)
	}

	withCause := crashed
	withCause.Cause = &Cause{ModKey: "k", ModName: "Named", Reason: "code", Detail: "first line\nsecond"}
	setRuns(withCause)
	h, err := svc.CrashHint(lc, p.ID)
	if err != nil || h == nil || h.RunID != "r1" || h.ModKey != "k" || h.ModName != "Named" || h.Reason != "first line" || h.Code != "code" {
		t.Fatalf("recorded cause: %+v %v", h, err)
	}

	setRuns(crashed)
	h, err = svc.CrashHint(lc, p.ID)
	if err != nil || h == nil || h.ModKey != modKey || h.ModName != "Fixture" || h.Reason == "" {
		t.Fatalf("log naming an enabled mod: %+v %v", h, err)
	}

	setLog(blamesStranger)
	h, err = svc.CrashHint(lc, p.ID)
	if err != nil || h == nil || h.ModKey != "" || h.Reason != "missing dependencies: Alpha" || h.Code != "missingDependencies" {
		t.Fatalf("finding naming no installed mod: %+v %v", h, err)
	}

	setLog(blamesStranger + blamesFixture)
	h, err = svc.CrashHint(lc, p.ID)
	if err != nil || h == nil || h.ModKey != modKey || h.Reason != "missing dependencies: Lib" {
		t.Fatalf("a later finding that names a mod beats an earlier one that does not: %+v %v", h, err)
	}

	setLog(header)
	h, err = svc.CrashHint(lc, p.ID)
	if err != nil || h == nil || h.RunID != "r1" || h.ModKey != "" || h.Reason != "" {
		t.Fatalf("empty log: %+v %v", h, err)
	}
}
