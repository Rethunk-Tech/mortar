package settings

import (
	"fmt"
	"maps"
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
	"tab-notes":         "Ctrl+6",
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

func rejectDuplicateShortcuts(s Settings) error {
	used := map[string]string{}
	for id, keys := range s.Shortcuts {
		if other, ok := used[keys]; ok {
			return fmt.Errorf("%s is already used by %s", keys, other)
		}
		used[keys] = id
	}
	return nil
}

func normalizeShortcuts(s *Settings) {
	next := DefaultShortcuts()
	for id, keys := range s.Shortcuts {
		if _, ok := defaultShortcuts[id]; ok && keys != "" {
			next[id] = keys
		}
	}
	s.Shortcuts = next
}
