package launchsvc

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gamesettings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

// field is one startup setting: how to set it on a Settings, and how to tell it is the only one set.
type field struct {
	name string
	// raw is a valid element value, and set stores a distinct one.
	raw string
	set func(*gamesettings.Settings, int)
	get func(gamesettings.Settings) bool
}

func intField(name string, ptr func(*gamesettings.Settings) **int) field {
	return field{
		name: name, raw: "5",
		set: func(s *gamesettings.Settings, n int) { *ptr(s) = &n },
		get: func(s gamesettings.Settings) bool { return *ptr(&s) != nil },
	}
}

func allFields() []field {
	return []field{
		{
			name: "windowMode", raw: "windowed",
			set: func(s *gamesettings.Settings, n int) {
				v := []string{"windowed", "fullscreen", "borderless"}[n%3]
				s.WindowMode = &v
			},
			get: func(s gamesettings.Settings) bool { return s.WindowMode != nil },
		},
		intField("displayIndex", func(s *gamesettings.Settings) **int { return &s.DisplayIndex }),
		intField("preferredResolutionX", func(s *gamesettings.Settings) **int { return &s.PreferredResolutionX }),
		intField("preferredResolutionY", func(s *gamesettings.Settings) **int { return &s.PreferredResolutionY }),
		intField("fullscreenResolutionX", func(s *gamesettings.Settings) **int { return &s.FullscreenResolutionX }),
		intField("fullscreenResolutionY", func(s *gamesettings.Settings) **int { return &s.FullscreenResolutionY }),
		intField("zoomLevel", func(s *gamesettings.Settings) **int { return &s.ZoomLevel }),
		intField("uiScale", func(s *gamesettings.Settings) **int { return &s.UIScale }),
		{
			name: "startMuted", raw: "true",
			set: func(s *gamesettings.Settings, n int) { v := n == 1; s.StartMuted = &v },
			get: func(s gamesettings.Settings) bool { return s.StartMuted != nil },
		},
		intField("musicVolumeLevel", func(s *gamesettings.Settings) **int { return &s.MusicVolumeLevel }),
		intField("soundVolumeLevel", func(s *gamesettings.Settings) **int { return &s.SoundVolumeLevel }),
	}
}

func only(f field, n int) gamesettings.Settings {
	var s gamesettings.Settings
	f.set(&s, n)
	return s
}

func everything(n int) gamesettings.Settings {
	var s gamesettings.Settings
	for _, f := range allFields() {
		f.set(&s, n)
	}
	return s
}

// onlyThis reports whether f is the one field s holds.
func onlyThis(s gamesettings.Settings, f field) bool {
	for _, other := range allFields() {
		if other.get(s) != (other.name == f.name) {
			return false
		}
	}
	return true
}

func TestEmptySettingsSeesEveryField(t *testing.T) {
	if !emptySettings(gamesettings.Settings{}) {
		t.Fatal("nothing set is not empty")
	}
	for _, f := range allFields() {
		if emptySettings(only(f, 1)) {
			t.Errorf("%s set but the settings read as empty", f.name)
		}
	}
}

func TestSettingNamesFlagsOnlySetFields(t *testing.T) {
	for _, f := range allFields() {
		for name, flagged := range settingNames(only(f, 1)) {
			if flagged != (name == f.name) {
				t.Errorf("with only %s set, %s flagged = %v", f.name, name, flagged)
			}
		}
	}
	if got := settingNames(gamesettings.Settings{}); len(got) != len(allFields()) {
		t.Fatalf("names = %v", got)
	}
}

func TestSettingsPresentKeepsOnlyWhatBothHold(t *testing.T) {
	for _, f := range allFields() {
		if got := settingsPresent(everything(1), only(f, 2)); !onlyThis(got, f) {
			t.Errorf("%s present in the file: kept %+v", f.name, got)
		}
		if got := settingsPresent(only(f, 1), gamesettings.Settings{}); !emptySettings(got) {
			t.Errorf("%s absent from the file but kept: %+v", f.name, got)
		}
		if got := settingsPresent(gamesettings.Settings{}, everything(1)); !emptySettings(got) {
			t.Errorf("%s not asked for but kept: %+v", f.name, got)
		}
		// The wanted value is kept, not the file's.
		if got := settingsPresent(only(f, 1), only(f, 2)); !onlyThis(got, f) || f.name != "windowMode" && f.name != "startMuted" && *intOf(got, f.name) != 1 {
			t.Errorf("%s: kept %+v", f.name, got)
		}
	}
}

func intOf(s gamesettings.Settings, name string) *int {
	return map[string]*int{
		"displayIndex": s.DisplayIndex, "preferredResolutionX": s.PreferredResolutionX,
		"preferredResolutionY": s.PreferredResolutionY, "fullscreenResolutionX": s.FullscreenResolutionX,
		"fullscreenResolutionY": s.FullscreenResolutionY, "zoomLevel": s.ZoomLevel, "uiScale": s.UIScale,
		"musicVolumeLevel": s.MusicVolumeLevel, "soundVolumeLevel": s.SoundVolumeLevel,
	}[name]
}

func TestSettingsUnchangedRestoresOnlyWhatTheGameLeftAlone(t *testing.T) {
	for _, f := range allFields() {
		written, original := everything(1), everything(2)
		// Every field but f was changed by the game since Mortar wrote it.
		current := everything(3)
		f.set(&current, 1)
		got := settingsUnchanged(current, written, original)
		if !onlyThis(got, f) {
			t.Errorf("%s untouched by the game: restored %+v", f.name, got)
		}
		if got := settingsUnchanged(everything(3), written, original); !emptySettings(got) {
			t.Errorf("%s: every field changed by the game but %+v is restored", f.name, got)
		}
		// A field the file no longer has, or one Mortar never wrote, is not restored.
		if got := settingsUnchanged(gamesettings.Settings{}, written, original); !emptySettings(got) {
			t.Errorf("a field gone from the file was restored: %+v", got)
		}
		if got := settingsUnchanged(written, gamesettings.Settings{}, original); !emptySettings(got) {
			t.Errorf("a field Mortar never wrote was restored: %+v", got)
		}
	}
}

func TestSetSettingValueParsesEachField(t *testing.T) {
	for _, f := range allFields() {
		var got gamesettings.Settings
		if err := setSettingValue(&got, f.name, f.raw); err != nil || !onlyThis(got, f) {
			t.Errorf("%s=%s: %+v, %v", f.name, f.raw, got, err)
		}
		if f.name == "windowMode" {
			continue
		}
		var bad gamesettings.Settings
		if err := setSettingValue(&bad, f.name, "not a value"); err == nil || !emptySettings(bad) {
			t.Errorf("%s accepted garbage: %+v, %v", f.name, bad, err)
		}
	}
	var unknown gamesettings.Settings
	if err := setSettingValue(&unknown, "somethingElse", "x"); err != nil || !emptySettings(unknown) {
		t.Fatalf("an unknown element: %+v, %v", unknown, err)
	}
}

func TestReadStartupSettingsOnlyReadsDirectChildrenOfStartupPreferences(t *testing.T) {
	data := []byte(`<?xml version="1.0"?>
<other><windowMode>borderless</windowMode></other>
<startup_preferences>
  <nested><displayIndex>9</displayIndex></nested>
  <windowMode> windowed </windowMode>
  <displayIndex>1</displayIndex>
  <ignored>3</ignored>
</startup_preferences>
<after><zoomLevel>40</zoomLevel></after>`)
	want := everything(1)
	got, err := readStartupSettings(data, want)
	if err != nil {
		t.Fatal(err)
	}
	if got.WindowMode == nil || *got.WindowMode != "windowed" || got.DisplayIndex == nil || *got.DisplayIndex != 1 {
		t.Fatalf("read %+v", got)
	}
	if got.ZoomLevel != nil {
		t.Fatalf("an element after the section was read: %+v", got)
	}
	// Only the settings Mortar wants are read.
	partial, err := readStartupSettings(data, only(allFields()[0], 1))
	if err != nil || partial.DisplayIndex != nil || partial.WindowMode == nil {
		t.Fatalf("partial = %+v, %v", partial, err)
	}
	// Nothing wanted: nothing to read, so even a broken file is not parsed.
	if empty, err := readStartupSettings([]byte("<<<"), gamesettings.Settings{}); err != nil || !emptySettings(empty) {
		t.Fatalf("empty = %+v, %v", empty, err)
	}
	if _, err := readStartupSettings([]byte("<startup_preferences><windowMode>x"), want); err == nil {
		t.Fatal("a truncated file was read as complete")
	}
	if _, err := readStartupSettings([]byte("<startup_preferences><displayIndex>wide</displayIndex></startup_preferences>"), want); err == nil {
		t.Fatal("a number that is not one was read")
	}
}

const prefsXML = `<startup_preferences>
  <windowMode>fullscreen</windowMode>
  <displayIndex>1</displayIndex>
  <soundVolumeLevel>80</soundVolumeLevel>
</startup_preferences>`

func prefsPath(config string) string {
	return filepath.Join(config, "StardewValley", "startup_preferences")
}

func writePrefs(t *testing.T, config, body string) string {
	t.Helper()
	path := prefsPath(config)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := fsx.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadStartupSettingsSectionBoundaries(t *testing.T) {
	want := everything(1)
	// A section inside the section does not shift which elements count as direct children.
	got, err := readStartupSettings([]byte(`<startup_preferences><startup_preferences><windowMode>borderless</windowMode></startup_preferences></startup_preferences>`), want)
	if err != nil || got.WindowMode != nil {
		t.Fatalf("a grandchild was read: %+v, %v", got, err)
	}
	// A deeper copy after the real value does not replace it.
	got, err = readStartupSettings([]byte(`<startup_preferences><displayIndex>1</displayIndex><x><displayIndex>9</displayIndex></x></startup_preferences>`), want)
	if err != nil || got.DisplayIndex == nil || *got.DisplayIndex != 1 {
		t.Fatalf("the nested copy won: %+v, %v", got, err)
	}
	// Two sections are both read, and what sits two levels down after them is not.
	got, err = readStartupSettings([]byte(`<startup_preferences><windowMode>windowed</windowMode></startup_preferences>
<startup_preferences><displayIndex>2</displayIndex></startup_preferences>
<wrap><inner><zoomLevel>50</zoomLevel></inner></wrap>`), want)
	if err != nil || got.WindowMode == nil || got.DisplayIndex == nil || *got.DisplayIndex != 2 || got.ZoomLevel != nil {
		t.Fatalf("sections = %+v, %v", got, err)
	}
}

func TestRecoverGoesOnPastProfilesItCannotUse(t *testing.T) {
	svc, first, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	second := testenv.Profile(t, svc.profiles, "stardew", "Second")
	broken := testenv.Profile(t, svc.profiles, "stardew", "Broken")
	third := testenv.Profile(t, svc.profiles, "stardew", "Third")
	dir, err := svc.profiles.ProfileDir("stardew", broken.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	mode := "windowed"
	if err := svc.SetGameSettings("stardew", third.ID, gamesettings.Settings{WindowMode: &mode}); err != nil {
		t.Fatal(err)
	}
	restore, _, err := svc.prepareGameSettings("stardew", third.ID, "")
	if err != nil || restore == nil {
		t.Fatal(err)
	}
	// The first profile has no record at all, and the second has one that cannot be read.
	settingsPath, _ := svc.profileSettingsPath("stardew", second.ID)
	if err := os.WriteFile(filepath.Join(filepath.Dir(settingsPath), settingsRestoreFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = first
	if err := svc.RecoverGameSettings(); err == nil {
		t.Fatal("the unreadable record was not reported")
	}
	if got := readBytes(t, path); !bytes.Equal(got, []byte(prefsXML)) {
		t.Fatalf("the later profile's record was not recovered:\n%s", got)
	}
}

// setAndPrepare saves a window mode for the profile and starts a launch, returning the record that undoes it.
func setAndPrepare(t *testing.T, svc *Service, profileID, mode string) *settingsRestore {
	t.Helper()
	if err := svc.SetGameSettings("stardew", profileID, gamesettings.Settings{WindowMode: &mode}); err != nil {
		t.Fatal(err)
	}
	restore, _, err := svc.prepareGameSettings("stardew", profileID, "")
	if err != nil || restore == nil {
		t.Fatalf("prepare: %v %v", restore, err)
	}
	return restore
}

func TestRestoreBringsBackTheExactBytes(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	mode, display := "windowed", 2
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{WindowMode: &mode, DisplayIndex: &display}); err != nil {
		t.Fatal(err)
	}
	restore, _, err := svc.prepareGameSettings("stardew", p.ID, "")
	if err != nil || restore == nil {
		t.Fatalf("prepare: %v %v", restore, err)
	}
	if bytes.Equal(readBytes(t, path), []byte(prefsXML)) {
		t.Fatal("prepare changed nothing")
	}
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, []byte(prefsXML)) {
		t.Fatalf("restored bytes differ:\n%s", got)
	}
}

func TestRestoreWritesNothingTheSecondTime(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	restore := setAndPrepare(t, svc, p.ID, "windowed")
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
	// The player changes the mode in the game's own menu afterwards.
	changed := bytes.Replace([]byte(prefsXML), []byte("fullscreen"), []byte("borderless"), 1)
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecoverGameSettings(); err != nil {
		t.Fatal(err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, changed) {
		t.Fatalf("a restore that already ran wrote again:\n%s", got)
	}
	if err := svc.restoreGameSettings(nil); err != nil {
		t.Fatalf("nothing to restore: %v", err)
	}
}

func TestPrepareNeverTouchesPrefsItCannotRead(t *testing.T) {
	for name, body := range map[string]string{
		"corrupt xml": "<startup_preferences><windowMode>fullscreen",
		"bad number":  "<startup_preferences><windowMode>fullscreen</windowMode><displayIndex>wide</displayIndex></startup_preferences>",
	} {
		svc, p, config := newGameSettingsService(t)
		path := writePrefs(t, config, body)
		mode, display := "windowed", 2
		if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{WindowMode: &mode, DisplayIndex: &display}); err != nil {
			t.Fatal(err)
		}
		restore, _, err := svc.prepareGameSettings("stardew", p.ID, "")
		if err == nil || restore != nil {
			t.Fatalf("%s: prepared %v, %v", name, restore, err)
		}
		if got := readBytes(t, path); string(got) != body {
			t.Fatalf("%s: prefs rewritten: %s", name, got)
		}
		settingsPath, _ := svc.profileSettingsPath("stardew", p.ID)
		if _, err := os.Stat(filepath.Join(filepath.Dir(settingsPath), settingsRestoreFile)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s: a restore record was left: %v", name, err)
		}
	}
}

func TestPrepareWritesNothingWhenThereIsNothingToChange(t *testing.T) {
	mode, absent := "fullscreen", 3
	for name, tc := range map[string]struct {
		body  string
		value gamesettings.Settings
	}{
		"already set":        {prefsXML, gamesettings.Settings{WindowMode: &mode}},
		"not in the file":    {prefsXML, gamesettings.Settings{ZoomLevel: &absent}},
		"overrides all gone": {prefsXML, gamesettings.Settings{}},
	} {
		svc, p, config := newGameSettingsService(t)
		path := writePrefs(t, config, tc.body)
		if err := svc.SetGameSettings("stardew", p.ID, tc.value); err != nil {
			t.Fatal(err)
		}
		restore, missing, err := svc.prepareGameSettings("stardew", p.ID, "")
		if err != nil || restore != nil || missing {
			t.Fatalf("%s: %v %v %v", name, restore, missing, err)
		}
		if got := readBytes(t, path); string(got) != tc.body {
			t.Fatalf("%s: prefs rewritten: %s", name, got)
		}
	}
}

func TestRestoreKeepsCorruptPrefsAndTheRecord(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	restore := setAndPrepare(t, svc, p.ID, "windowed")
	// The game crashed halfway through writing its preferences.
	torn := "<startup_preferences><windowMode>windowed"
	if err := os.WriteFile(path, []byte(torn), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.restoreGameSettings(restore); err == nil {
		t.Fatal("a restore over corrupt prefs reported success")
	}
	if got := readBytes(t, path); string(got) != torn {
		t.Fatalf("corrupt prefs were overwritten: %s", got)
	}
	if _, err := os.Stat(restore.recordPath); err != nil {
		t.Fatalf("the record was dropped, so the restore can never be retried: %v", err)
	}
}

func TestRestoreLeavesPrefsTheGameChangedEverywhere(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	restore := setAndPrepare(t, svc, p.ID, "windowed")
	changed := bytes.Replace(readBytes(t, path), []byte("windowed"), []byte("borderless"), 1)
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, changed) {
		t.Fatalf("a value the player changed in game was reverted: %s", got)
	}
	if _, err := os.Stat(restore.recordPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the record outlived a finished restore: %v", err)
	}
}

func TestRestoreWithPrefsAlreadyAsTheyWereDropsTheRecord(t *testing.T) {
	svc, p, config := newGameSettingsService(t)
	path := writePrefs(t, config, prefsXML)
	restore := setAndPrepare(t, svc, p.ID, "windowed")
	// The game rewrote the file back to the original on its own.
	if err := os.WriteFile(path, []byte(prefsXML), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.restoreGameSettings(restore); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(restore.recordPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the record outlived a restore with nothing left to do: %v", err)
	}
}

func TestRecoverSkipsProfilesWithoutARecordAndReportsABadOne(t *testing.T) {
	svc, p, _ := newGameSettingsService(t)
	if err := svc.RecoverGameSettings(); err != nil {
		t.Fatalf("no records: %v", err)
	}
	settingsPath, _ := svc.profileSettingsPath("stardew", p.ID)
	record := filepath.Join(filepath.Dir(settingsPath), settingsRestoreFile)
	if err := os.WriteFile(record, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecoverGameSettings(); err == nil {
		t.Fatal("an unreadable record was ignored")
	}
}

func TestGameSettingsRoundTripAndMissingFile(t *testing.T) {
	svc, p, _ := newGameSettingsService(t)
	if got, err := svc.GameSettings("stardew", p.ID); err != nil || !emptySettings(got) {
		t.Fatalf("no settings yet: %+v %v", got, err)
	}
	display := 4
	if err := svc.SetGameSettings("stardew", p.ID, gamesettings.Settings{DisplayIndex: &display}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GameSettings("stardew", p.ID)
	if err != nil || got.DisplayIndex == nil || *got.DisplayIndex != 4 {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	path, _ := svc.profileSettingsPath("stardew", p.ID)
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GameSettings("stardew", p.ID); err == nil {
		t.Fatal("broken settings read as empty")
	}
}
