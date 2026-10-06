package nexus

import "testing"

func TestParseModURL(t *testing.T) {
	for _, tc := range []struct {
		in     string
		domain string
		id     int
		ok     bool
	}{
		{"https://www.nexusmods.com/stardewvalley/mods/1915", "stardewvalley", 1915, true},
		{"https://www.nexusmods.com/stardewvalley/mods/1915?tab=files", "stardewvalley", 1915, true},
		{"https://nexusmods.com/games/stardewvalley/mods/1915/", "stardewvalley", 1915, true},
		{"https://next.nexusmods.com/stardewvalley/mods/1915#x", "stardewvalley", 1915, true},
		{"https://www.nexusmods.com/stardewvalley/mods/1915/files", "", 0, false},
		{"https://www.nexusmods.com/games/stardewvalley/collections/tckf0m", "", 0, false},
		{"https://www.nexusmods.com/stardewvalley/mods/0", "", 0, false},
		{"https://www.nexusmods.com/stardewvalley/mods/012", "", 0, false},
		{"http://www.nexusmods.com/stardewvalley/mods/1915", "", 0, false},
		{"https://example.com/stardewvalley/mods/1915", "", 0, false},
	} {
		domain, id, ok := ParseModURL(tc.in)
		if ok != tc.ok || domain != tc.domain || id != tc.id {
			t.Errorf("%s: got %q %d %v", tc.in, domain, id, ok)
		}
	}
}
