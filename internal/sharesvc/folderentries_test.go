package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

func TestDescribeCountsASplitArchiveOnceAndListsEveryFile(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	entry := func(file string) profile.Entry {
		k := "pkg-1#" + file
		return profile.Entry{Key: k, Item: "pkg-1", File: file, Source: src, Package: true,
			Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, k), Name: file, Folder: "."}}}
	}
	info, err := describe("sims4", profile.Profile{Name: "P", Entries: []profile.Entry{entry("a.package"), entry("b.package")}}, profile.ShareFacts{})
	if err != nil || info.Count != 1 || len(info.Groups) != 1 || len(info.Groups[0].Mods) != 2 {
		t.Fatalf("%+v %v", info, err)
	}
}

func TestDescribeCountsAKeptWholeArchiveWithTrayFilesOnce(t *testing.T) {
	t.Parallel()
	src := profile.Source{Kind: profile.KindNexus, ModID: 5, FileID: 9}
	whole := profile.Entry{Key: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, "pkg-1"), Name: "Mod", Folder: "."}}}
	tray := profile.Entry{Key: "pkg-1#tray", Item: "pkg-1", Source: src, Package: true, Mods: []profile.Component{{ID: mod.NewID(mod.FormatFolder, "pkg-1#tray"), Name: "Tray", Folder: "."}}}
	info, err := describe("sims4", profile.Profile{Name: "P", Entries: []profile.Entry{whole, tray}}, profile.ShareFacts{})
	if err != nil || info.Count != 1 {
		t.Fatalf("%+v %v", info, err)
	}
}
