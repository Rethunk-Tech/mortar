package problems

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func (fakeMeta) PageRequirements(context.Context, int) ([]meta.Requirement, error) {
	return nil, nil
}

func (fakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return meta.Collection{}, nil
}

// CheckUpdates is checkUpdates without a fresh lookup or Nexus file listing.
func CheckUpdates(ctx context.Context, m Meta, env Environment, mods []Installed, enabledOnly bool) UpdatesResult {
	return checkUpdates(ctx, m, env, mods, enabledOnly, false, nil)
}
