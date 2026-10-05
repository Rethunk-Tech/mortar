package all_test

import (
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
)

func TestForGameFollowsCatalogOrderAndSkipsUnregistered(t *testing.T) {
	t.Parallel()
	g := components.GameInfo{Sources: []components.GameSource{{ID: "curseforge"}, {ID: "nexus"}, {ID: "github"}}}
	var got []string
	for _, s := range source.ForGame(g) {
		got = append(got, s.ID())
	}
	if want := []string{"nexus", "github"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	var searchable []string
	for _, s := range source.Searchable(g) {
		searchable = append(searchable, s.ID())
	}
	if !slices.Equal(searchable, []string{"nexus", "github"}) {
		t.Fatalf("searchable %v", searchable)
	}
}

func TestSchemesAndHosts(t *testing.T) {
	t.Parallel()
	if !source.IsLink("NXM://stardewvalley/mods/1/files/2") || source.IsLink("https://x") {
		t.Fatal("nxm should be the only claimed scheme")
	}
	if name, ok := source.NameOfHost("www.Itch.io"); !ok || name != "itch.io" {
		t.Fatalf("host lookup %q %v", name, ok)
	}
}

func TestOptInSchemeIsClaimedOnlyWhenChosen(t *testing.T) {
	if source.IsLink("ror2mm://v1/install/thunderstore.io/A/B/1.0.0/") {
		t.Fatal("ror2mm claimed by default")
	}
	source.SetHandleLinks(map[string]bool{"thunderstore": true})
	t.Cleanup(func() { source.SetHandleLinks(nil) })
	if !source.IsLink("ror2mm://v1/install/thunderstore.io/A/B/1.0.0/") || !source.IsLink("nxm://x/mods/1/files/2") {
		t.Fatal("opted-in ror2mm and default nxm should both be claimed")
	}
}
