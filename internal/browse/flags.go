package browse

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

// thunderstoreLoaderPack is the BepInEx loader's Thunderstore package, which Mortar's loader install supplies.
const thunderstoreLoaderPack = "BepInEx-BepInExPack"

// mark sets each hit's Obsolete, Broken and Loader flags. The compatibility list is read only when the game has
// one, and a failure to read it leaves nothing marked broken.
func (c *Client) mark(ctx context.Context, info components.GameInfo, items []Item) {
	var idx meta.CompatIndex
	if c.Compat != nil {
		idx, _ = c.Compat(ctx)
	}
	for i := range items {
		it := &items[i]
		it.Obsolete = it.Obsolete || problems.ObsoleteText(it.Name, it.Summary)
		switch it.Source {
		case "nexus":
			id, err := strconv.Atoi(it.ID)
			if err != nil {
				continue
			}
			it.Broken = idx.ByNexus[id].Status == meta.StatusBroken
			it.Loader = slices.ContainsFunc(info.Loaders, func(l components.GameLoader) bool { return l.NexusModID == id })
		case "thunderstore":
			it.Loader = strings.EqualFold(it.ID, thunderstoreLoaderPack) &&
				slices.ContainsFunc(info.Loaders, func(l components.GameLoader) bool { return strings.HasPrefix(l.ID, "bepinex") })
		}
	}
}

// applyModes drops the hits the filter hides and counts them in Hidden.
func (c *Client) applyModes(p *Page, f Filter) {
	before := len(p.Items)
	p.Items = slices.DeleteFunc(p.Items, func(it Item) bool {
		return (f.Installed == ModeHide && it.Installed) ||
			(f.Obsolete == ModeHide && it.Obsolete) ||
			(f.Broken == ModeHide && it.Broken)
	})
	p.Hidden = before - len(p.Items)
}
