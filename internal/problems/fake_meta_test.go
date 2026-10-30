package problems

import (
	"context"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
)

func (fakeMeta) Collection(context.Context, string, string, int) (nexusmods.Collection, error) {
	return nexusmods.Collection{}, nil
}

// CheckUpdates is checkUpdates without a fresh lookup or Nexus file listing.
func CheckUpdates(ctx context.Context, m Meta, env Environment, mods []framework.Mod, enabledOnly bool) UpdatesResult {
	return checkUpdates(ctx, m, env, mods, enabledOnly, false, nil)
}

var testEnv = Environment{Nexus: nexusmods.Title{Domain: "stardewvalley", ID: 1303}, VersionScheme: deps.SemverSMAPI}
