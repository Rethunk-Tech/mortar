package launchsvc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/launch"
)

func TestBisectFailsOnlyOnACrash(t *testing.T) {
	if bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "These mods could not be added to your game."}}) {
		t.Fatal("a skipped mod is not a crash; counting it blames whichever mod is skipped")
	}
	if bisectStartupFailure([]launch.Entry{{Level: launch.Error, Message: "Steam achievements won't work"}}) {
		t.Fatal("non-fatal Steam warning was treated as a crash")
	}
	if !bisectStartupFailure([]launch.Entry{{Level: launch.Alert, Message: "fatal startup error"}}) {
		t.Fatal("SMAPI crash report was not detected")
	}
	if !summaryHealthy(launch.Summary{Errors: 5}) {
		t.Fatal("a run with errors but no crash must count as healthy")
	}
	if summaryHealthy(launch.Summary{Crashed: true}) {
		t.Fatal("a crashed run must not count as healthy")
	}
}

func TestStartupReportAfterIgnoresOlderFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2020-01-01T00-00-00.000Z.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(time.Second)
	if startupReportAfter(dir, old) {
		t.Fatal("an existing report from before launch is not a title-screen signal")
	}
	if err := os.Chtimes(path, old.Add(time.Second), old.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if !startupReportAfter(dir, old) {
		t.Fatal("a report newer than launch must count as healthy")
	}
}

func TestProfileHasBridgeNeedsTheFolder(t *testing.T) {
	dir := t.TempDir()
	if profileHasBridge(dir) {
		t.Fatal("empty mods dir has no bridge")
	}
	if err := os.MkdirAll(filepath.Join(dir, "bridge-1.3.0-f129f1a70915", bridge.ModFolder), 0o700); err != nil {
		t.Fatal(err)
	}
	if !profileHasBridge(dir) {
		t.Fatal("bridge folder was not seen")
	}
}
