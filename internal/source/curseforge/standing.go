package curseforge

import (
	"context"
	"strconv"

	cf "github.com/Rethunk-Tech/go-curseforge"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// standingBatch is how many mods one request names.
const standingBatch = 500

// Standings flags the mods CurseForge marks Inactive, Abandoned or Deleted, or no longer available, one request per
// five hundred mods. The answer is keyed by the id as given.
func (d Driver) Standings(ctx context.Context, _ string, ids []string) (map[string]source.Standing, error) {
	out := map[string]source.Standing{}
	for start := 0; start < len(ids); start += standingBatch {
		var asked []int
		for _, id := range ids[start:min(start+standingBatch, len(ids))] {
			if n, err := strconv.Atoi(id); err == nil {
				asked = append(asked, n)
			}
		}
		if len(asked) == 0 {
			continue
		}
		mods, err := call(d, func(c *cf.Client) ([]cf.Mod, error) { return c.Mods(ctx, asked) })
		if err != nil {
			return nil, err
		}
		for _, m := range mods {
			id := strconv.Itoa(m.ID)
			switch {
			case m.Status == cf.StatusDeleted:
				out[id] = source.Standing{State: "removed"}
			case m.Status == cf.StatusAbandoned:
				out[id] = source.Standing{State: "abandoned"}
			case m.Status == cf.StatusInactive:
				out[id] = source.Standing{State: "inactive"}
			case !m.Available():
				out[id] = source.Standing{State: "unavailable"}
			}
		}
	}
	return out, nil
}
