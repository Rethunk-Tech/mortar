package browse

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
)

func TestMarkAndHide(t *testing.T) {
	info := components.GameInfo{Loaders: []components.GameLoader{{ID: "smapi", NexusModID: 2400}}}
	c := &Client{Compat: func(context.Context) (meta.CompatIndex, error) {
		return meta.CompatIndex{ByNexus: map[int]meta.CompatEntry{7: {Status: meta.StatusBroken}, 8: {Status: meta.StatusOK}}}, nil
	}}
	items := []Item{
		{Source: "nexus", ID: "2400", Name: "SMAPI"},
		{Source: "nexus", ID: "7", Name: "Old Mod (deprecated)"},
		{Source: "nexus", ID: "8", Name: "Fine", Installed: true},
		{Source: "nexus", ID: "9", Name: "Plain", Summary: "Replaces the obsolete X."},
	}
	c.mark(t.Context(), info, items)
	if !items[0].Loader || items[1].Loader || !items[1].Broken || !items[1].Obsolete || items[2].Broken || items[3].Obsolete {
		t.Fatalf("flags %+v", items)
	}
	p := Page{Total: 4, Items: items}
	c.applyModes(&p, Filter{Installed: ModeHide, Broken: ModeHide, Obsolete: ModeGray})
	if len(p.Items) != 2 || p.Hidden != 2 || p.Total != 4 {
		t.Fatalf("hidden %d of %+v", p.Hidden, p.Items)
	}
}
