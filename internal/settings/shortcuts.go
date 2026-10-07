package settings

import (
	"fmt"
	"maps"
	"slices"
)

// DefaultShortcuts is the chord table Settings › Shortcuts lists when nothing is rebound; it matches SHORTCUTS in
// frontend/src/settings/shortcuts.ts, which shortcuts.test.ts there checks.
func DefaultShortcuts() map[string]string {
	return maps.Clone(defaultShortcuts)
}

var defaultShortcuts = map[string]string{
	"command-palette":   "Ctrl+K",
	"filter-mods":       "Ctrl+F",
	"play":              "Ctrl+P",
	"check-updates":     "F5",
	"open-settings":     "Ctrl+,",
	"dismiss":           "Esc",
	"select-all-mods":   "Ctrl+A",
	"mod-up":            "↑",
	"mod-down":          "↓",
	"mod-toggle":        "Space",
	"mod-details":       "Enter",
	"mod-remove":        "Delete",
	"tab-browse":        "Ctrl+1",
	"tab-mods":          "Ctrl+2",
	"tab-problems":      "Ctrl+3",
	"tab-load-order":    "Ctrl+4",
	"tab-saves":         "Ctrl+5",
	"tab-config":        "Ctrl+6",
	"tab-console":       "Ctrl+7",
	"tab-performance":   "Ctrl+8",
	"new-profile":       "Ctrl+N",
	"duplicate-profile": "Ctrl+D",
	"rename-profile":    "F2",
	"find-all-mods":     "Ctrl+Shift+F",
	"import":            "Ctrl+I",
	"export-profile":    "Ctrl+E",
	"downloads":         "Ctrl+J",
	"notifications":     "Ctrl+Shift+N",
	"previous-profile":  "Ctrl+PageUp",
	"next-profile":      "Ctrl+PageDown",
	"collapse-sidebar":  "Ctrl+B",
	"back":              "Alt+Left",
	"help":              "F1",
	"vanilla-play":      "Ctrl+Shift+P",
}

func rejectUnknownShortcuts(s Settings) error {
	for id := range s.Shortcuts {
		if _, ok := defaultShortcuts[id]; !ok {
			return fmt.Errorf("unknown shortcut %q", id)
		}
	}
	return nil
}

// formerDefaults are chords earlier builds shipped as defaults and wrote into settings.json; one found there is that
// build's default, not something the user chose.
var formerDefaults = map[string][]string{
	"tab-mods":        {"Ctrl+1"},
	"tab-problems":    {"Ctrl+2"},
	"tab-saves":       {"Ctrl+3"},
	"tab-console":     {"Ctrl+5"},
	"tab-performance": {"Ctrl+6"},
}

// rejectDuplicateShortcuts refuses two actions on one chord; an unbound action ("") clashes with nothing.
func rejectDuplicateShortcuts(s Settings) error {
	used := map[string]string{}
	for id, keys := range s.Shortcuts {
		if keys == "" {
			continue
		}
		if other, ok := used[keys]; ok {
			return fmt.Errorf("%s is already used by %s", keys, other)
		}
		used[keys] = id
	}
	return nil
}

// userShortcuts are the chords the user chose: known ids bound to something other than a default, now or before.
func userShortcuts(chords map[string]string) map[string]string {
	user := map[string]string{}
	for id, keys := range chords {
		def, known := defaultShortcuts[id]
		if known && keys != "" && keys != def && !slices.Contains(formerDefaults[id], keys) {
			user[id] = keys
		}
	}
	return user
}

// withShortcutOverrides keeps only the user's chords, so a changed default reaches everyone who never rebound it.
func withShortcutOverrides(s Settings) Settings {
	s.Shortcuts = userShortcuts(s.Shortcuts)
	return s
}

// normalizeShortcuts lays the user's chords over the defaults; a default the user took for another action is left
// unbound, so the user's choice wins.
func normalizeShortcuts(s *Settings) {
	user := userShortcuts(s.Shortcuts)
	taken := map[string]bool{}
	for _, keys := range user {
		taken[keys] = true
	}
	next := DefaultShortcuts()
	for id, keys := range next {
		if taken[keys] {
			next[id] = ""
		}
	}
	maps.Copy(next, user)
	s.Shortcuts = next
}
