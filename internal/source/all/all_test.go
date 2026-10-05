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
	g := components.GameInfo{Sources: []components.GameSource{{ID: "moddrop"}, {ID: "thunderstore"}, {ID: "nexus"}, {ID: "github"}}}
	var got []string
	for _, s := range source.ForGame(g) {
		got = append(got, s.ID())
	}
	if want := []string{"moddrop", "nexus", "github"}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := source.Searchable(g); !slices.Equal(got, []string{"nexus", "github"}) {
		t.Fatalf("searchable %v", got)
	}
}

func TestSchemesAndHosts(t *testing.T) {
	t.Parallel()
	if !source.IsLink("NXM://stardewvalley/mods/1/files/2") || source.IsLink("https://x") {
		t.Fatal("nxm should be the only claimed scheme")
	}
	if name, ok := source.NameOfHost("www.ModDrop.com"); !ok || name != "ModDrop" {
		t.Fatalf("host lookup %q %v", name, ok)
	}
}

func TestModDropLinkUsesCatalogKey(t *testing.T) {
	t.Parallel()
	e, _ := source.Get("moddrop")
	linker, ok := e.Source.(source.PageLinker)
	if !ok {
		t.Fatal("moddrop should link pages")
	}
	if got := linker.ModPageURL("stardew-valley", 42); got != "https://www.moddrop.com/stardew-valley/mods/42" {
		t.Fatalf("got %s", got)
	}
}
