package settings

import (
	"encoding/json"
	"maps"
	"os"
	"strings"
	"testing"
)

func TestSetShortcutsPersistsAndFillsDefaults(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	got := s.Get().Shortcuts
	if got["downloads"] != "Ctrl+J" || got["vanilla-play"] != "Ctrl+Shift+P" {
		t.Fatalf("defaults = %#v", got)
	}
	if err := svc.SetShortcuts(map[string]string{"downloads": "Ctrl+U"}); err != nil {
		t.Fatal(err)
	}
	got = s.Get().Shortcuts
	if got["downloads"] != "Ctrl+U" || got["play"] != "Ctrl+P" {
		t.Fatalf("merged = %#v", got)
	}
}

func TestSetShortcutsRefusesUnknownAndTwoActionsOnOneChord(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	if err := svc.SetShortcuts(map[string]string{"play": "Ctrl+U", "downloads": "Ctrl+U"}); err == nil {
		t.Fatal("two actions on Ctrl+U")
	}
	if err := svc.SetShortcuts(map[string]string{"nope": "Ctrl+9"}); err == nil {
		t.Fatal("unknown id")
	}
	if s.Get().Shortcuts["play"] != "Ctrl+P" {
		t.Fatalf("kept defaults after refusal: %#v", s.Get().Shortcuts)
	}
}

func TestAUserChordTakesItFromTheDefaultAction(t *testing.T) {
	s, _ := open(t)
	if err := NewService(s).SetShortcuts(map[string]string{"play": "Ctrl+K"}); err != nil {
		t.Fatal(err)
	}
	got := s.Get().Shortcuts
	if got["play"] != "Ctrl+K" || got["command-palette"] != "" {
		t.Fatalf("play = %q, command-palette = %q", got["play"], got["command-palette"])
	}
}

func TestAUserChordSurvivesAReopen(t *testing.T) {
	s, _ := open(t)
	if err := NewService(s).SetShortcuts(map[string]string{"downloads": "Ctrl+U"}); err != nil {
		t.Fatal(err)
	}
	again, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Get().Shortcuts["downloads"]; got != "Ctrl+U" {
		t.Fatalf("downloads = %q", got)
	}
}

func TestSettingsFileKeepsOnlyReboundShortcuts(t *testing.T) {
	s, _ := open(t)
	if err := NewService(s).SetShortcuts(map[string]string{"downloads": "Ctrl+U"}); err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), `"downloads": "Ctrl+U"`) || strings.Contains(string(doc), `"play": "Ctrl+P"`) {
		t.Fatalf("settings file = %s", doc)
	}
}

// An earlier build wrote its whole default table, old tab chords included, into settings.json.
func TestAFileHoldingAnOldDefaultTableLoadsSavesAndStoresNothing(t *testing.T) {
	s, _ := open(t)
	old := DefaultShortcuts()
	maps.Copy(old, map[string]string{
		"tab-mods": "Ctrl+1", "tab-problems": "Ctrl+2", "tab-saves": "Ctrl+3",
		"tab-console": "Ctrl+5", "tab-performance": "Ctrl+6",
	})
	delete(old, "tab-browse")
	delete(old, "tab-load-order")
	doc, err := json.Marshal(map[string]any{"formatVersion": 1, "global": map[string]any{"shortcuts": old}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path, doc, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Get().Shortcuts; !maps.Equal(got, DefaultShortcuts()) {
		t.Fatalf("shortcuts = %#v", got)
	}
	if _, err := loaded.Update(func(v *Settings) { v.Theme = ThemeLight }); err != nil {
		t.Fatalf("an unrelated save was refused: %v", err)
	}
	saved, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "Ctrl+") {
		t.Fatalf("settings file still stores chords: %s", saved)
	}
}
