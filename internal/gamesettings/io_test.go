package gamesettings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	mode, mute, vol := "borderless", true, 40
	want := Settings{WindowMode: &mode, StartMuted: &mute, MusicVolumeLevel: &vol}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Load = %+v, %v, want %+v", got, err, want)
	}
}

func TestSaveAndLoadRefuseBadSettings(t *testing.T) {
	dir := t.TempDir()
	neg := -1
	path := filepath.Join(dir, "s.json")
	if err := Save(path, Settings{DisplayIndex: &neg}); err == nil {
		t.Error("Save accepted a negative display index")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Save wrote a file for invalid settings")
	}
	if _, err := Load(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("Load of a missing file succeeded")
	}
	for name, body := range map[string]string{"junk.json": "{", "range.json": `{"uiScale":101}`} {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(p); err == nil {
			t.Errorf("Load accepted %s", name)
		}
	}
}
