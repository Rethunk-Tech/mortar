// Package metadata maps the catalog's per-game metadata ids onto the providers that answer them.
package metadata

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// Updates answers "is there a newer version" for installed mods.
type Updates interface {
	CheckUpdates(ctx context.Context, req meta.UpdateRequest) []meta.UpdateResult
}

// Status answers which mods are known broken.
type Status interface {
	CompatList(ctx context.Context) (meta.CompatIndex, error)
}

// Dataset maps a mod's id to its pages on the download sites.
type Dataset interface {
	Lookup(ctx context.Context, uniqueID string) ([]meta.Ref, error)
	Index(ctx context.Context) (map[string][]meta.Ref, error)
}

// Providers is what a game's metadata ids resolve to; a capability the game lacks is nil.
type Providers struct {
	Updates Updates
	Status  Status
	Dataset Dataset
}

// Catalog ids of the built-in providers.
const (
	SMAPIUpdates   = "smapi-updates"
	SMAPICompat    = "smapi-compat"
	StardewDataset = "stardew-dataset"
)

// For resolves a game's catalog metadata ids against c. Unknown ids are ignored.
func For(c *meta.Client, ids []string) Providers {
	var p Providers
	for _, id := range ids {
		switch id {
		case SMAPIUpdates:
			p.Updates = c
		case SMAPICompat:
			p.Status = c
		case StardewDataset:
			p.Dataset = c
		}
	}
	return p
}
