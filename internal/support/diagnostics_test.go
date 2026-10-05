package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func TestDiagnosticsRedactsSecrets(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	home := t.TempDir()

	apiKey := strings.Repeat("k", 24)
	nxmKey := strings.Repeat("n", 20)
	const (
		notes   = "do-not-include-notes"
		modName = "Content Patcher"
		modVer  = "2.1.0"
	)

	gameFolder := filepath.Join(home, "Games", "Stardew Valley")
	bg := filepath.Join(home, "Pictures", "wall.png")
	settingsIn := map[string]any{
		"accent":            "sand",
		"nexusName":         "FixtureUser",
		"nexusUserId":       87654321,
		"nexusPremium":      true,
		"nexusKey":          apiKey,
		"apiKey":            apiKey,
		"backgroundImage":   bg,
		"gameFolders":       map[string]string{"stardew": gameFolder},
		"lastPlayed":        map[string]any{"stardew": map[string]any{"profile": "Cozy Farm", "at": "2026-09-30T12:00:00Z"}},
		"overlayToken":      "overlay-secret-token",
		"nexusRefreshToken": "refresh-secret",
		"keyringItem":       "keyring-secret-item",
		"nested":            map[string]any{"clientSecret": "nested-secret"},
	}
	rawSettings, err := json.Marshal(settingsIn)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "settings.json"), rawSettings, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "queue.json"), []byte(`{
		"items": [{"id": "i1", "name": "A mod", "key": "`+nxmKey+`", "expires": 99, "nxmKey": "`+nxmKey+`"}],
		"paused": false,
		"limitedUntil": 0
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	profDir := filepath.Join(data, "profiles", "stardew", "p1")
	if err := os.MkdirAll(profDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profDir, "profile.json"), []byte(`{
		"id": "p1",
		"name": "Farm",
		"notes": "do-not-include-notes",
		"entries": [{
			"source": {"kind": "nexus", "name": "Content Patcher", "modId": 1915},
			"mods": [{"name": "Content Patcher", "id": "smapi:Pathoschild.ContentPatcher", "version": "2.1.0"}]
		}]
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, body := range map[string]string{
		"mortar.log":      "line one " + home + "/x\n",
		"mortar.prev.log": "prev-line " + home + "/y\n",
		"crash.log":       "crash-line\n",
	} {
		if err := os.WriteFile(filepath.Join(data, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runsDir := filepath.Join(profDir, "runs")
	if err := os.MkdirAll(runsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	var runs []string
	for i := 1; i <= 4; i++ {
		runs = append(runs, fmt.Sprintf(`{"id":"r%d","started":"2026-10-0%dT10:00:00Z","outcome":"ok"}`, i, i))
	}
	if err := os.WriteFile(filepath.Join(runsDir, "index.json"), []byte(`{"runs":[`+strings.Join(runs, ",")+`]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	logDir := filepath.Join(cfg, "StardewValley", "ErrorLogs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatal(err)
	}
	smapi := "[05:00:01 INFO  SMAPI] Mods go here: " + filepath.Join(home, "p", "a", "mods") + "\nsmapi-fixture-line\n"
	if err := os.WriteFile(filepath.Join(logDir, "SMAPI-latest.txt"), []byte(smapi), 0o600); err != nil {
		t.Fatal(err)
	}

	s := NewService("1.2.3", func(string) problems.Environment { return problems.Environment{} }, home,
		func(_, id string) (string, error) { return filepath.Join(home, "p", id, "mods"), nil })
	s.Dir = func() (string, error) { return data, nil }
	s.RecentLog = func(string, string) string { return "in-memory-console-line" }
	var saved []byte
	s.SaveZip = func(_, filename string, blob []byte) (string, error) {
		path := filepath.Join(t.TempDir(), filename)
		if err := os.WriteFile(path, blob, 0o600); err != nil {
			return "", err
		}
		saved = blob
		return path, nil
	}

	path, err := s.SaveDiagnostics("stardew", "a")
	if err != nil {
		t.Fatal(err)
	}
	if path == "" || saved == nil {
		t.Fatal("expected a zip to be written")
	}

	target := filepath.Join(t.TempDir(), "out.zip")
	if got, err := s.WriteDiagnostics("stardew", "a", target); err != nil || got != target {
		t.Fatalf("WriteDiagnostics = %q, %v", got, err)
	}
	written, err := fsx.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := zipNames(t, written)["settings.json"]; !ok {
		t.Fatal("the path-based zip lacks settings.json")
	}
	if _, err := s.WriteDiagnostics("stardew", "a", filepath.Join(t.TempDir(), "out.txt")); err == nil {
		t.Fatal("a non-zip path was accepted")
	}

	files := zipNames(t, saved)
	blob := files["settings.json"] + files["profiles.json"] + files["queue.json"] + files["manifest.txt"] + files["build.json"] + files["smapi-latest.txt"] + files["mortar.log"]
	for _, name := range []string{"mortar.log", "mortar.prev.log", "crash.log", "doctor.txt", "runs.json", "manifest.txt"} {
		if _, ok := files[name]; !ok {
			t.Errorf("zip lacks %s", name)
		}
		blob += files[name]
	}
	if strings.Contains(files["runs.json"], `"r1"`) || !strings.Contains(files["runs.json"], `"r4"`) {
		t.Errorf("runs.json should hold the 3 newest runs: %s", files["runs.json"])
	}
	if strings.Contains(blob, home) {
		t.Error("zip still contains the home folder")
	}
	for _, secret := range []string{"refresh-secret", "keyring-secret-item", "nested-secret", apiKey, nxmKey, notes, "FixtureUser", "overlay-secret-token"} {
		if strings.Contains(blob, secret) {
			t.Errorf("zip still contains %q", secret)
		}
	}
	if strings.Contains(blob, "87654321") {
		t.Error("zip still contains the Nexus user id")
	}
	if !strings.Contains(files["profiles.json"], modName) || !strings.Contains(files["profiles.json"], modVer) {
		t.Errorf("profiles.json omitted mod name or version: %s", files["profiles.json"])
	}
	if !strings.Contains(files["smapi-latest.txt"], "smapi-fixture-line") {
		t.Errorf("smapi log: %q", files["smapi-latest.txt"])
	}
	if !strings.Contains(files["mortar.log"], "line one ~/x") {
		t.Errorf("mortar.log: %q", files["mortar.log"])
	}
	if !strings.Contains(files["manifest.txt"], "Removed") || !strings.Contains(files["manifest.txt"], "Included") {
		t.Error("manifest.txt missing included/removed lists")
	}
	if !strings.Contains(files["build.json"], `"mortar": "1.2.3"`) {
		t.Errorf("build.json: %s", files["build.json"])
	}

	var settings map[string]any
	if err := json.Unmarshal([]byte(files["settings.json"]), &settings); err != nil {
		t.Fatal(err)
	}
	if settings["nexusName"] != "" {
		t.Errorf("nexusName %v", settings["nexusName"])
	}
	if settings["nexusUserId"] != float64(0) {
		t.Errorf("nexusUserId %v", settings["nexusUserId"])
	}
	if _, ok := settings["nexusPremium"]; ok {
		t.Error("nexusPremium should be removed")
	}
	if _, ok := settings["nexusKey"]; ok {
		t.Error("nexusKey key should be removed")
	}
	if _, ok := settings["overlayToken"]; ok {
		t.Error("overlayToken should be removed")
	}
	if settings["backgroundImage"] != filepath.Join("~", "Pictures", "wall.png") &&
		settings["backgroundImage"] != "~/Pictures/wall.png" {
		t.Errorf("backgroundImage %v", settings["backgroundImage"])
	}
	folders, _ := settings["gameFolders"].(map[string]any)
	gotFolder, _ := folders["stardew"].(string)
	if gotFolder != filepath.Join("~", "Games", "Stardew Valley") && gotFolder != "~/Games/Stardew Valley" {
		t.Errorf("gameFolders.stardew %q", gotFolder)
	}
	played, _ := settings["lastPlayed"].(map[string]any)
	entry, _ := played["stardew"].(map[string]any)
	if entry["profile"] != "Cozy Farm" {
		t.Errorf("lastPlayed profile %v", entry["profile"])
	}
	if _, ok := settings["accent"]; !ok {
		t.Error("settings lost accent")
	}

	var queue map[string]any
	if err := json.Unmarshal([]byte(files["queue.json"]), &queue); err != nil {
		t.Fatal(err)
	}
	items, _ := queue["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("queue items %v", queue["items"])
	}
	item, _ := items[0].(map[string]any)
	if _, ok := item["key"]; ok {
		t.Error("queue item still has key")
	}
	if _, ok := item["nxmKey"]; ok {
		t.Error("queue item still has nxmKey")
	}
	if item["name"] != "A mod" {
		t.Errorf("queue name %v", item["name"])
	}
}

func zipNames(t *testing.T, data []byte) map[string]string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			_ = rc.Close()
			t.Fatal(err)
		}
		_ = rc.Close()
		out[f.Name] = buf.String()
	}
	return out
}

func TestDiagnosticsIncludeTheLoaderLogOfAnyGame(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, "p", "lobby")
	if err := os.MkdirAll(filepath.Join(profile, "BepInEx"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profile, "BepInEx", "LogOutput.log"), []byte("bepinex-fixture-line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := NewService("1.2.3", func(string) problems.Environment { return problems.Environment{} }, home,
		func(_, id string) (string, error) { return filepath.Join(home, "p", id, "mods"), nil })
	name, body := s.loaderTail("lethal-company", "lobby")
	if name != "logoutput.log" || !strings.Contains(body, "bepinex-fixture-line") {
		t.Fatalf("loaderTail = %q, %q", name, body)
	}
	if name, body := s.loaderTail("lethal-company", "other"); body != "" {
		t.Fatalf("another profile's log: %q %q", name, body)
	}
}
