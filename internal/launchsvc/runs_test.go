package launchsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func runEnv(t *testing.T) (*Service, profile.Profile, string, string) {
	t.Helper()
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("APPDATA", cfg)
	home := t.TempDir()
	datadirtest.Use(t, filepath.Join(home, "data"))
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "A")
	return NewService(home, nil, profiles), p, cfg, home
}

func TestMissingFileCauseNamesKyuyaPack(t *testing.T) {
	modsDir := filepath.Join(t.TempDir(), "mods")
	path := filepath.Join(modsDir, "nexus-11780-74182", "[FS]Kyuya??s hats Pack", "assets", "hat.png")
	got, ok := missingFileCause([]profile.Mod{{Key: "nexus-11780-74182", Name: "[FS]Kyuya's hats Pack", ID: "smapi:Kyuya.Hats"}}, modsDir,
		[]string{fmt.Sprintf("ContentLoadException: Could not find a part of the path '%s'", path)})
	if !ok {
		t.Fatal("missing-file cause was not classified")
	}
	if got.ModName != "[FS]Kyuya's hats Pack" || got.Reason != "missing-file" ||
		got.Detail != "[FS]Kyuya's hats Pack: a file it needs could not be opened. Reinstall it." {
		t.Fatalf("cause = %#v", got)
	}
}

func writeOwnedLog(t *testing.T, cfg, home, modsDir, extra string) {
	t.Helper()
	dir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(home, modsDir)
	if err != nil {
		t.Fatal(err)
	}
	body := "SMAPI 4.5.2 with Stardew Valley 1.6.15 build 24356 on Unix\n" +
		"[19:43:46 INFO  SMAPI] Mods go here: ~/" + rel + "\n" + extra
	if err := os.WriteFile(filepath.Join(dir, "SMAPI-latest.txt"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRecordStoresOwnedLogAndBoundsHistory(t *testing.T) {
	svc, p, cfg, home := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	g := game.Find("stardew")
	writeOwnedLog(t, cfg, home, mods, "[19:43:50 ERROR Content Patcher] boom\n"+
		"[19:43:51 WARN  Pet Against Crows] Possible conflicts with this mod detected.\n")
	started := time.Now().Add(-2 * time.Second)
	svc.record(g, p.ID, started, false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %v, %v", runs, err)
	}
	got := runs[0]
	if got.Outcome != launch.OutcomeRan || got.LoaderVersion != "4.5.2" || got.GameVersion != "1.6.15" {
		t.Fatalf("meta = %#v", got)
	}
	if got.Errors != 1 || got.Warnings != 1 || got.Unclassified != 1 || got.DurationMs < 1000 {
		t.Fatalf("counts = %#v", got)
	}
	text, err := svc.RunLog("stardew", p.ID, got.ID)
	if err != nil || !strings.Contains(text, "Content Patcher") || !strings.Contains(text, "Mods go here:") {
		t.Fatalf("log = %q, %v", text, err)
	}
	issues, err := svc.LastRunIssues("stardew", p.ID)
	if err != nil || issues.RunID != got.ID || len(issues.Mods) != 0 {
		t.Fatalf("no installed user mod must map; issues = %#v, %v", issues, err)
	}
	lines, err := svc.RunLines("stardew", p.ID, got.ID)
	if err != nil || len(lines) == 0 {
		t.Fatalf("lines = %v, %v", lines, err)
	}

	other, err := svc.profiles.Create("stardew", "B")
	if err != nil {
		t.Fatal(err)
	}
	if runs, err := svc.Runs("stardew", other.ID); err != nil || len(runs) != 0 {
		t.Fatalf("other profile: %v, %v", runs, err)
	}

	for range maxRuns {
		writeOwnedLog(t, cfg, home, mods, "[19:43:50 ERROR Content Patcher] n\n")
		svc.record(g, p.ID, time.Now(), false)
	}
	runs, err = svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != maxRuns {
		t.Fatalf("bounded = %d, %v", len(runs), err)
	}
	dir := runsDir(mods)
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	txt := 0
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".txt") {
			txt++
		}
	}
	if txt != maxRuns {
		t.Fatalf("log files = %d", txt)
	}
}

func TestLastRunSummaryIncludesModSnapshot(t *testing.T) {
	svc, p, cfg, home := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeOwnedLog(t, cfg, home, mods, "[19:43:50 ERROR Alpha] old error\n")
	g := game.Find("stardew")
	ref := launch.ModRef{Name: "Alpha", ID: "smapi:me.mod", Key: "old-key", Version: "1.0", SourceVersion: "1.0"}
	svc.record(g, p.ID, time.Now(), false, []launch.ModRef{ref})
	_, summary, err := svc.LastRunSummary("stardew", p.ID)
	if err != nil || len(summary.ModRefs) != 1 || summary.ModRefs[0] != ref {
		t.Fatalf("summary refs = %#v, %v", summary.ModRefs, err)
	}
}

func TestRecordFailedLaunchAndUnownedLogUsesSession(t *testing.T) {
	svc, p, cfg, _ := runEnv(t)
	g := game.Find("stardew")
	dir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SMAPI-latest.txt"), []byte("[19:43:46 INFO  SMAPI] Mods go here: ~/someone-else\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	buf := &launch.Buffer{}
	buf.Add(launch.Entry{Time: "19:43:50", Level: launch.Error, Mod: "Farm", Message: "broke"})
	svc.logs["stardew/"] = session{buf: buf, profile: p.ID}
	svc.record(g, p.ID, time.Now(), true)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Outcome != launch.OutcomeFailed {
		t.Fatalf("failed = %#v, %v", runs, err)
	}
	text, err := svc.RunLog("stardew", p.ID, runs[0].ID)
	if err != nil || !strings.Contains(text, "broke") || strings.Contains(text, "someone-else") {
		t.Fatalf("must copy the session, not the other profile's log: %q, %v", text, err)
	}
}

func TestClosedRunWithCrashMarksCrashed(t *testing.T) {
	svc, p, cfg, home := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeOwnedLog(t, cfg, home, mods, "[19:43:51 ALERT SMAPI] The game crashed: boom\n")
	g := game.Find("stardew")
	svc.record(g, p.ID, time.Now(), false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Outcome != launch.OutcomeCrashed {
		t.Fatalf("crashed = %#v, %v", runs, err)
	}
}

func TestSearchRuns(t *testing.T) {
	svc, p, _, _ := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := runsDir(mods)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	index := runIndex{Runs: []Run{
		{ID: "new", Started: "2026-10-02T12:00:00Z", Outcome: launch.OutcomeFailed},
		{ID: "old", Started: "2026-10-01T12:00:00Z", Outcome: launch.OutcomeRan},
	}}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), mustJSON(t, index), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("nothing\nTarget NEW\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "old.txt"), []byte("target old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := svc.SearchRuns("stardew", p.ID, "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	if got.Truncated || len(got.Hits) != 2 || got.Hits[0].RunID != "new" || got.Hits[0].LineNumber != 2 ||
		got.Hits[0].Line != "Target NEW" || got.Hits[1].RunID != "old" {
		t.Fatalf("search = %#v", got)
	}
	empty, err := svc.SearchRuns("stardew", p.ID, "  ")
	if err != nil || len(empty.Hits) != 0 {
		t.Fatalf("empty search = %#v, %v", empty, err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRecordKeepsThePresetName(t *testing.T) {
	svc, p, cfg, home := runEnv(t)
	mods, err := svc.profiles.ModsDir("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	writeOwnedLog(t, cfg, home, mods, "")
	svc.mu.Lock()
	svc.logs["stardew/"] = session{profile: p.ID, preset: "Debug"}
	svc.mu.Unlock()
	svc.record(game.Find("stardew"), p.ID, time.Now(), false)
	runs, err := svc.Runs("stardew", p.ID)
	if err != nil || len(runs) != 1 || runs[0].Preset != "Debug" {
		t.Fatalf("runs = %+v, %v", runs, err)
	}
}

func TestRecordStoresTheBepInExLogOfTheProfile(t *testing.T) {
	datadirtest.Use(t, filepath.Join(t.TempDir(), "data"))
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "lethal-company", "A")
	dir, err := profiles.ProfileDir("lethal-company", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	logDir := filepath.Join(dir, "BepInEx")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	const body = "[Info   :   BepInEx] BepInEx 5.4.22 - Lethal Company\n[Error  :   BepInEx] Error loading [Foo]: boom\n"
	if err := os.WriteFile(filepath.Join(logDir, "LogOutput.log"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, profiles)
	svc.record(game.Find("lethal-company"), p.ID, time.Now(), false)
	runs, err := svc.Runs("lethal-company", p.ID)
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %v, %v", runs, err)
	}
	if got, err := svc.RunLog("lethal-company", p.ID, runs[0].ID); err != nil || got != body {
		t.Fatalf("stored log = %q, %v", got, err)
	}
}

func TestUnityCrashMarkers(t *testing.T) {
	for text, want := range map[string]bool{
		"Crash!!!\nSymbolInfo:": true, "Fatal error in GC": true, "Exception: x\n  at Foo": false,
	} {
		if unityCrashed(text) != want {
			t.Errorf("unityCrashed(%q) = %v", text, !want)
		}
	}
}
