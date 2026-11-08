package launchsvc

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
)

// crashDumpService is a Lethal Company profile with one mod and a crashed run whose CatLib crash folder is the testdata
// fixture named, laid out where a real BepInEx launch leaves it. It returns the service, the profile and that folder.
func crashDumpService(t *testing.T, fixture string) (*Service, string, string) {
	t.Helper()
	set, ps := testenv.Env(t)
	const lc = "lethal-company"
	p := testenv.Profile(t, ps, lc, "A")
	zip := testfs.WriteZip(t, filepath.Join(t.TempDir(), "BetterRepair.zip"), map[string]string{
		"manifest.json":            `{"name":"BetterRepair","version_number":"1.0.0"}`,
		"plugins/BetterRepair.dll": "MZ",
	})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, zip, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-BetterRepair", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	profileDir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	modsDir, err := ps.ModsDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(runsDir(modsDir), 0o700); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	if err := datadir.WriteJSON(filepath.Join(runsDir(modsDir), "index.json"), runIndex{
		Runs: []Run{{ID: "r1", Started: started, Outcome: launch.OutcomeCrashed}},
	}); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(profileDir, "BepInEx", "CatLib", "Crashes", "2026-11-08_15-00-00_4242")
	copyTree(t, filepath.Join("testdata", "catlib", fixture), reportDir)
	svc := NewService(t.TempDir(), set, ps)
	// A symbol server names a frame only when it holds the module's PDB, which a fixture dump does not.
	svc.Symbols = nil
	return svc, p.ID, reportDir
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o700); err != nil {
		t.Fatal(err)
	}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		b, err := safefs.ReadFile(path)
		if err != nil {
			return err
		}
		return safefs.WriteFile(filepath.Join(dst, rel), b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A crash dump that faulted inside a mod's own DLL blames that mod, names the frame the game died in and points the
// window at the crash folder the mod library kept.
func TestCrashHintBlamesTheModADumpDiedIn(t *testing.T) {
	svc, profileID, reportDir := crashDumpService(t, "mod-fault")
	hint, err := svc.CrashHint("lethal-company", profileID)
	if err != nil || hint == nil {
		t.Fatalf("hint = %+v, %v", hint, err)
	}
	if hint.ModKey == "" || hint.ModName != "BetterRepair" || hint.Code != "modCode" || hint.Reason != "The game crashed inside this mod's code." {
		t.Fatalf("hint blamed %q %q (%q) %q", hint.ModKey, hint.ModName, hint.Code, hint.Reason)
	}
	if hint.NativeFrame != "BetterRepair.dll+0x1c" || hint.ReportDir != reportDir {
		t.Fatalf("frame %q report %q, want %q", hint.NativeFrame, hint.ReportDir, reportDir)
	}
	if !strings.Contains(hint.Evidence, "UnityPlayer.dll") {
		t.Fatalf("the report's caller is missing from %q", hint.Evidence)
	}
}

// A dump whose fault is in the game's own executable names no mod: the hint says the game crashed and still points at
// the folder the crash report is in.
func TestCrashHintCallsADumpWithNoModFrameAGameCrash(t *testing.T) {
	svc, profileID, reportDir := crashDumpService(t, "game-dump")
	hint, err := svc.CrashHint("lethal-company", profileID)
	if err != nil || hint == nil {
		t.Fatalf("hint = %+v, %v", hint, err)
	}
	if hint.ModKey != "" || hint.ModName != "" {
		t.Fatalf("a game crash blamed %q %q", hint.ModKey, hint.ModName)
	}
	if hint.Code != "gameCrash" || hint.Params["module"] != "Lethal Company.exe" ||
		hint.Reason != "The game itself crashed (Lethal Company.exe), not a mod's code." {
		t.Fatalf("reason %q %q %v", hint.Reason, hint.Code, hint.Params)
	}
	if hint.NativeFrame != "Lethal Company.exe+0x42" || hint.ReportDir != reportDir {
		t.Fatalf("frame %q report %q, want %q", hint.NativeFrame, hint.ReportDir, reportDir)
	}
}

// A run that left no crash folder has nothing to show, so OpenCrashReport says so instead of opening a path.
func TestOpenCrashReportWithoutACrash(t *testing.T) {
	set, ps := testenv.Env(t)
	p := testenv.Profile(t, ps, "lethal-company", "A")
	svc := NewService(t.TempDir(), set, ps)
	if err := svc.OpenCrashReport("lethal-company", p.ID); err == nil || err.Error() != "the last run left no crash report" {
		t.Fatalf("err = %v", err)
	}
}
