package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/problems"
)

func TestDiagnosticsRedactsSecrets(t *testing.T) {
	data := t.TempDir()
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	home := t.TempDir()

	const (
		apiKey  = "test-nexus-key-not-real"
		userID  = 87654321
		nxmKey  = "nxm-fixture-key-xyz"
		notes   = "do-not-include-notes"
		modName = "Content Patcher"
		modVer  = "2.1.0"
	)

	if err := os.WriteFile(filepath.Join(data, "settings.json"), []byte(`{
		"accent": "sand",
		"nexusName": "FixtureUser",
		"nexusUserId": 87654321,
		"nexusKey": "test-nexus-key-not-real",
		"apiKey": "test-nexus-key-not-real"
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "queue.json"), []byte(`{
		"items": [{"id": "i1", "name": "A mod", "key": "nxm-fixture-key-xyz", "expires": 99, "nxmKey": "nxm-fixture-key-xyz"}],
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
			"mods": [{"name": "Content Patcher", "uniqueId": "Pathoschild.ContentPatcher", "version": "2.1.0"}]
		}]
	}`), 0o600); err != nil {
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

	files := zipNames(t, saved)
	blob := files["settings.json"] + files["profiles.json"] + files["queue.json"] + files["README.txt"] + files["build.json"] + files["smapi-latest.txt"] + files["console.txt"]
	for _, secret := range []string{apiKey, nxmKey, notes, "FixtureUser"} {
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
	if !strings.Contains(files["console.txt"], "in-memory-console-line") {
		t.Errorf("console: %q", files["console.txt"])
	}
	if !strings.Contains(files["README.txt"], "Removed") || !strings.Contains(files["README.txt"], "Included") {
		t.Error("README.txt missing included/removed lists")
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
	if _, ok := settings["nexusKey"]; ok {
		t.Error("nexusKey key should be removed")
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
