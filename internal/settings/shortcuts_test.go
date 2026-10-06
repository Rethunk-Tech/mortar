package settings

import (
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

func TestSetShortcutsRefusesConflictAndUnknown(t *testing.T) {
	s, _ := open(t)
	svc := NewService(s)
	if err := svc.SetShortcuts(map[string]string{"play": "Ctrl+K"}); err == nil {
		t.Fatal("conflict with command-palette")
	}
	if err := svc.SetShortcuts(map[string]string{"nope": "Ctrl+9"}); err == nil {
		t.Fatal("unknown id")
	}
	if s.Get().Shortcuts["play"] != "Ctrl+P" {
		t.Fatalf("kept defaults after refusal: %#v", s.Get().Shortcuts)
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

func TestAClashingShortcutFileStillSavesOtherSettings(t *testing.T) {
	s, _ := open(t)
	s.cur.Shortcuts["tab-mods"] = "Ctrl+1"
	if _, err := s.Update(func(v *Settings) { v.Theme = ThemeLight }); err != nil {
		t.Fatalf("an unrelated save was refused: %v", err)
	}
}
