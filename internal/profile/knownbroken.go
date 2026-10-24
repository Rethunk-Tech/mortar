package profile

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// withKnownBroken marks each previewed mod the game's known-broken list flags at its version.
func withKnownBroken(ctx context.Context, game string) func(GameModsPreview, error) (GameModsPreview, error) {
	return func(pv GameModsPreview, err error) (GameModsPreview, error) {
		if err != nil {
			return pv, err
		}
		list := components.KnownBrokenFor(ctx, game)
		for i, m := range pv.Mods {
			if e, ok := list.Find(components.BrokenMod{ID: m.ID, NexusID: m.NexusModID, Version: m.Version}, ""); ok {
				pv.Mods[i].Broken, pv.Mods[i].BrokenKind = e.Reason, e.Status()
			}
		}
		return pv, nil
	}
}
