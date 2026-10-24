package browse

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// mark sets each hit's Obsolete, Broken and Loader flags. The compatibility list is read only when the game has
// one, and a failure to read it leaves nothing marked broken.
func (c *Client) mark(ctx context.Context, info components.GameInfo, items []Item) {
	var idx meta.CompatIndex
	if c.Compat != nil {
		idx, _ = c.Compat(ctx)
	}
	known := components.KnownBrokenFor(ctx, info.ID)
	for i := range items {
		it := &items[i]
		it.Obsolete = it.Obsolete || problems.ObsoleteText(it.Name, it.Summary)
		if e, listed := known.Find(brokenModOf(*it), ""); listed {
			it.KnownBroken = &source.KnownBrokenNote{Kind: e.Status(), Reason: e.Reason, Replacement: e.Replacement}
			switch e.Status() {
			case components.BrokenKindBroken:
				it.Broken = true
			case components.BrokenKindSuperseded:
				it.Obsolete = true
			}
		}
		switch it.Source {
		case "nexus":
			id, err := strconv.Atoi(it.ID)
			if err != nil {
				continue
			}
			it.Broken = it.Broken || idx.ByNexus[id].Status == meta.StatusBroken
			it.Loader = slices.ContainsFunc(info.Loaders, func(l components.GameLoader) bool { return l.NexusModID == id })
		case "thunderstore":
			it.Loader = strings.EqualFold(it.ID, info.LoaderPackage()) &&
				slices.ContainsFunc(info.Loaders, func(l components.GameLoader) bool { return strings.HasPrefix(l.ID, "bepinex") })
		}
	}
}

// brokenModOf is a hit as a known-broken list names it: its id on its site, its repository, and the newest version,
// which is what Browse offers.
func brokenModOf(it Item) components.BrokenMod {
	m := components.BrokenMod{GitHub: it.Repo, Version: it.Version}
	switch it.Source {
	case "thunderstore":
		m.ID = mod.NewID(mod.FormatThunderstore, it.ID)
	case "nexus":
		m.NexusID, _ = strconv.Atoi(it.ID)
	case "curseforge":
		m.CurseForge, _ = strconv.Atoi(it.ID)
	case "github":
		m.GitHub = it.ID
	}
	return m
}

// holdLoader counts the loader with this id as held: the profile runs it although no entry names its mod page.
func (h Holdings) holdLoader(info components.GameInfo, loaderID string) {
	for _, l := range info.Loaders {
		if l.ID != loaderID {
			continue
		}
		if l.NexusModID != 0 {
			h.nexus[l.NexusModID] = true
		}
		if strings.HasPrefix(l.ID, "bepinex") {
			h.hold(profile.KindThunderstore, info.LoaderPackage())
		}
	}
}

// applyModes drops the hits the filter hides and adds them to Hidden, which may already count the source's own. Mortar's own bridges and the game's loader are
// dropped uncounted: Mortar installs those itself.
func (c *Client) applyModes(p *Page, f Filter) {
	p.Items = slices.DeleteFunc(p.Items, func(it Item) bool { return it.Loader || bridge.IsOwnListing(it.Name) })
	before := len(p.Items)
	p.Items = slices.DeleteFunc(p.Items, func(it Item) bool {
		return (f.Installed == ModeHide && it.Installed) ||
			(f.Obsolete == ModeHide && it.Obsolete) ||
			(f.Broken == ModeHide && it.Broken)
	})
	p.Hidden += before - len(p.Items)
}
