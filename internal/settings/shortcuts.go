package settings

import (
	"fmt"
	"maps"
)

// DefaultShortcuts is the chord table Settings › Shortcuts lists when nothing is rebound.
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
	"tab-mods":          "Ctrl+1",
	"tab-problems":      "Ctrl+2",
	"tab-saves":         "Ctrl+3",
	"tab-notes":         "Ctrl+4",
	"tab-console":       "Ctrl+5",
	"tab-performance":   "Ctrl+6",
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
