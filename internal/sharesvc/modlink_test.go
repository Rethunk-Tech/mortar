package sharesvc

import (
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

func TestParseModPageURL(t *testing.T) {
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
		domain, id, ok := parseModPageURL(tc.in)
		if ok != tc.ok || domain != tc.domain || id != tc.id {
			t.Errorf("%s: got %q %d %v", tc.in, domain, id, ok)
		}
	}
}

func TestPickModPageFile(t *testing.T) {
	old, now := time.Unix(100, 0), time.Unix(200, 0)
	if f := pickModPageFile([]nexus.File{{FileID: 1, Category: "MAIN"}, {FileID: 2, IsPrimary: true}}); f == nil || f.FileID != 2 {
		t.Fatalf("primary: %+v", f)
	}
	if f := pickModPageFile([]nexus.File{{FileID: 1, Category: "MAIN", Uploaded: old}, {FileID: 3, Category: "MAIN", Uploaded: now}, {FileID: 4, Category: "OPTIONAL", Uploaded: now}}); f == nil || f.FileID != 3 {
		t.Fatalf("newest main: %+v", f)
	}
	if f := pickModPageFile([]nexus.File{{FileID: 4, Category: "OPTIONAL"}}); f != nil {
		t.Fatalf("optional only: %+v", f)
	}
}
