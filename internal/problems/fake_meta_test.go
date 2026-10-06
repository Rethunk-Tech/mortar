package problems

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
)

func (fakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return meta.Collection{}, nil
}

// CheckUpdates is checkUpdates without a fresh lookup or Nexus file listing.
func CheckUpdates(ctx context.Context, m Meta, env Environment, mods []framework.Mod, enabledOnly bool) UpdatesResult {
	return checkUpdates(ctx, m, env, mods, enabledOnly, false, nil)
}

var testEnv = Environment{Nexus: nexus.Title{Domain: "stardewvalley", ID: 1303}, VersionScheme: deps.SemverSMAPI}
