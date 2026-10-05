package launchsvc

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gamesettings"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func newGameSettingsService(t *testing.T) (*Service, profile.Profile, string) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("APPDATA", config)

	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	p := testenv.Profile(t, profiles, "stardew", "Settings")
	return NewService("", set, profiles), p, config
}

func TestGameSettingsRestoreKeepsGameChanges(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	startup := filepath.Join(config, "StardewValley", "startup_preferences")
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

	restore, missing, err := svc.prepareGameSettings("stardew", p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if missing || restore == nil {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	if _, err := os.Stat(restore.recordPath); err != nil {
		t.Fatalf("restore record was not persisted: %v", err)
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
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
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
	if _, err := os.Stat(restore.recordPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restore record survived: %v", err)
	}
}

func TestGameSettingsLeftoverRestoreRunsAtStartup(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	startup := filepath.Join(config, "StardewValley", "startup_preferences")
	if err := os.MkdirAll(filepath.Dir(startup), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(startup, []byte(`<startup_preferences><windowMode>fullscreen</windowMode></startup_preferences>`), 0o600); err != nil {
		t.Fatal(err)
	}
	mode := "windowed"
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{WindowMode: &mode}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.prepareGameSettings("stardew", p.ID, ""); err != nil {
		t.Fatal(err)
	}
	svc2 := NewService("", nil, svc.profiles)
	if err := svc2.RecoverGameSettings(); err != nil {
		t.Fatal(err)
	}
	got, err := fsx.ReadFile(startup)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("<windowMode>fullscreen</windowMode>")) {
		t.Fatalf("leftover settings were not restored: %s", got)
	}
}

func TestGameSettingsRestoreReportsReadError(t *testing.T) {
	svc, _, _ := newGameSettingsService(t)
	err := svc.restoreGameSettings(&settingsRestore{
		path:       filepath.Join(t.TempDir(), "missing"),
		recordPath: filepath.Join(t.TempDir(), "record"),
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restore error = %v, want missing startup preferences", err)
	}
}

func TestGameSettingsMissingStartupPreferencesIsSkipped(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	mode := "windowed"
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{WindowMode: &mode}); err != nil {
		t.Fatal(err)
	}
	restore, missing, err := svc.prepareGameSettings("stardew", p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if restore != nil || !missing {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	if _, err := os.Stat(filepath.Join(config, "StardewValley", "startup_preferences")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing startup preferences was created: %v", err)
	}
}

func TestGameSettingsWithoutOverridesDoesNotTouchStartupPreferences(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	restore, missing, err := svc.prepareGameSettings("stardew", p.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if restore != nil || missing {
		t.Fatalf("prepare returned restore %v, missing %v", restore, missing)
	}
	if _, err := os.Stat(filepath.Join(config, "StardewValley", "startup_preferences")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile without overrides touched startup preferences: %v", err)
	}
}
