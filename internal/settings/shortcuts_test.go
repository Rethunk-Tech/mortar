package settings

import "testing"

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
