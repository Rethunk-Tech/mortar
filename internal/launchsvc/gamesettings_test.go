package launchsvc

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/gamesettings"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/store"
)

func newGameSettingsService(t *testing.T) (*Service, profile.Profile, string) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)

	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	p, err := profiles.Create("stardew", "Settings")
	if err != nil {
		t.Fatal(err)
	}
	return NewService("", nil, profiles), p, config
}

func TestGameSettingsRestoreKeepsGameChanges(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	startup := filepath.Join(config, "StardewValley", startupPreferencesFile)
	if err := os.MkdirAll(filepath.Dir(startup), 0o700); err != nil {
		t.Fatal(err)
	}
	initial := []byte(`<startup_preferences>
<windowMode>fullscreen</windowMode>
<displayIndex>1</displayIndex>
<soundVolumeLevel>80</soundVolumeLevel>
</startup_preferences>`)
	if err := os.WriteFile(startup, initial, 0o600); err != nil {
		t.Fatal(err)
	}
	mode := "windowed"
	display := 2
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{
		WindowMode:   &mode,
		DisplayIndex: &display,
	}); err != nil {
		t.Fatal(err)
	}

	restore, missing, err := svc.prepareGameSettings("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if missing || restore == nil {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	patched, err := fsx.ReadFile(startup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(patched, []byte("<windowMode>windowed</windowMode>")) ||
		!bytes.Contains(patched, []byte("<displayIndex>2</displayIndex>")) {
		t.Fatalf("startup preferences were not patched: %s", patched)
	}

	gameChanged := bytes.Replace(patched, []byte("<displayIndex>2</displayIndex>"), []byte("<displayIndex>7</displayIndex>"), 1)
	if err := os.WriteFile(startup, gameChanged, 0o600); err != nil {
		t.Fatal(err)
	}
	svc.restoreGameSettings(restore)
	got, err := fsx.ReadFile(startup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("<windowMode>fullscreen</windowMode>")) {
		t.Fatalf("Mortar value was not restored: %s", got)
	}
	if !bytes.Contains(got, []byte("<displayIndex>7</displayIndex>")) {
		t.Fatalf("game change was overwritten: %s", got)
	}
	if !bytes.Contains(got, []byte("<soundVolumeLevel>80</soundVolumeLevel>")) {
		t.Fatalf("untouched setting changed: %s", got)
	}
}

func TestGameSettingsMissingStartupPreferencesIsSkipped(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	mode := "windowed"
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{WindowMode: &mode}); err != nil {
		t.Fatal(err)
	}
	restore, missing, err := svc.prepareGameSettings("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restore != nil || !missing {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	if _, err := os.Stat(filepath.Join(config, "StardewValley", startupPreferencesFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing startup preferences was created: %v", err)
	}
}

func TestGameSettingsWithoutOverridesDoesNotTouchStartupPreferences(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	restore, missing, err := svc.prepareGameSettings("stardew", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restore != nil || missing {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	if _, err := os.Stat(filepath.Join(config, "StardewValley", startupPreferencesFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile without overrides touched startup preferences: %v", err)
	}
}
