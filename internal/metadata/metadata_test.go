package metadata

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestForResolvesByID(t *testing.T) {
	c := &meta.Client{}
	p := For(c, []string{SMAPICompat, "unknown"})
	if p.Status == nil || p.Updates != nil || p.Dataset != nil {
		t.Fatalf("got %+v", p)
	}
	all := For(c, []string{SMAPIUpdates, SMAPICompat, StardewDataset})
	if all.Updates == nil || all.Status == nil || all.Dataset == nil {
		t.Fatalf("got %+v", all)
	}
	if got := For(c, nil); got != (Providers{}) {
		t.Fatalf("empty ids: %+v", got)
	}
}

func TestCatalogIDsAreRegistered(t *testing.T) {
	g, ok := components.BundledGameByNexusDomain("stardewvalley")
	if !ok {
		t.Fatal("no stardew in catalog")
	}
	p := For(&meta.Client{}, g.Metadata)
	if p.Updates == nil || p.Status == nil || p.Dataset == nil {
		t.Fatalf("stardew metadata %v resolved to %+v", g.Metadata, p)
	}
}
