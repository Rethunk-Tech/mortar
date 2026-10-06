package launchsvc

import (
	"os"
	"path/filepath"
	"strings"
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
	if err := os.MkdirAll(filepath.Join(dir, "bridge-1.3.0-f129f1a70915", bridge.SMAPI.ModFolder), 0o700); err != nil {
		t.Fatal(err)
	}
	if !profileHasBridge(dir) {
		t.Fatal("bridge folder was not seen")
	}
}

func TestABepInExBisectStepWaitsForTheGameToSettleInAScene(t *testing.T) {
	log := func(lines ...string) []launch.Entry { return launch.ParseLog(strings.Join(lines, "\n")) }
	const (
		ready = "[Message:   BepInEx] Chainloader startup complete"
		first = "[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene InitSceneLaunchOptions: True"
		menu  = "[Info   :Mortar BepInEx Bridge] Bridge plugin alive after scene MainMenu: True"
	)
	t0 := time.Now()
	var w sceneWatch
	if w.settled(log("[Info   :   BepInEx] Loading [Foo 1.0.0]"), t0.Add(time.Minute)) {
		t.Fatal("a chainloader still loading plugins is not healthy, however long it takes")
	}
	if w.settled(log(ready, first), t0) || w.settled(log(ready, first), t0.Add(bisectSceneSettle)) {
		t.Fatal("the first scene, where Lethal Company waits for a click, needs the longer grace")
	}
	if w.settled(log(ready, first, menu), t0.Add(bisectSceneSettle+time.Second)) {
		t.Fatal("a scene that has just loaded is where mods crash; it is not healthy yet")
	}
	if !w.settled(log(ready, first, menu), t0.Add(2*bisectSceneSettle+time.Second)) {
		t.Fatal("a game that held its menu scene is healthy")
	}

	var bare sceneWatch
	if bare.settled(log(ready), t0) || !bare.settled(log(ready), t0.Add(bisectStartupGrace)) {
		t.Fatal("without the bridge the chainloader finishing starts the grace")
	}
}
