package browse

import (
	"context"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/profile"
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

func TestAnInstalledLoaderIsInstalledAndHidden(t *testing.T) {
	info := components.GameInfo{Loaders: []components.GameLoader{{ID: "smapi", NexusModID: 2400}, {ID: "bepinex5"}}}
	for _, tc := range []struct {
		loader string
		hit    Item
	}{
		{"smapi", Item{Source: "nexus", ID: "2400", Name: "SMAPI"}},
		{"bepinex5", Item{Source: "thunderstore", ID: "BepInEx-BepInExPack", Name: "BepInExPack"}},
	} {
		h := Hold(info, profile.Profile{}, nil)
		h.holdLoader(info, tc.loader)
		c := &Client{Installed: h.Has, Bundled: h.Bundled}
		items := []Item{tc.hit, {Source: "nexus", ID: "9", Name: "Plain"}}
		c.markInstalled(items)
		c.mark(t.Context(), info, items)
		if !items[0].Loader || !items[0].Installed || items[1].Installed {
			t.Fatalf("%s: flags %+v", tc.loader, items)
		}
		p := Page{Total: 2, Items: items}
		c.applyModes(&p, Filter{Installed: ModeHide})
		if len(p.Items) != 1 || p.Items[0].ID != "9" {
			t.Fatalf("%s: shown %+v", tc.loader, p.Items)
		}
	}
}
