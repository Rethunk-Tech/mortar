package share

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// A kept-whole archive that also lays out Tray files is two entries: the whole archive (no Item) and its tray entry
// (Item set). The link must name the archive once.
func TestCollectNamesAKeptWholeArchiveWithTrayFilesOnce(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	whole := profile.Entry{Key: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, "pkg-1")}}}
	tray := profile.Entry{Key: "pkg-1#tray", Item: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, "pkg-1#tray")}}}
	s, _, _ := Collect(profile.Profile{Entries: []profile.Entry{whole, tray}})
	if len(s.Entries) != 1 {
		t.Fatalf("the archive appears %d times in the link", len(s.Entries))
	}
}
