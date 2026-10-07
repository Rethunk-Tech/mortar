package curseforge

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// standingBatch is how many mods one request names.
const standingBatch = 500

// CurseForge's ModStatus values that mean the mod is no longer maintained or gone.
const (
	statusInactive  = 7
	statusAbandoned = 8
	statusDeleted   = 9
)

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
		var resp struct {
			Data []struct {
				ID          int   `json:"id"`
				Status      int   `json:"status"`
				IsAvailable *bool `json:"isAvailable"`
			} `json:"data"`
		}
		if err := d.do(ctx, http.MethodPost, "/mods", nil, map[string]any{"modIds": asked}, &resp); err != nil {
			return nil, err
		}
		for _, m := range resp.Data {
			id := strconv.Itoa(m.ID)
			switch {
			case m.Status == statusDeleted:
				out[id] = source.Standing{State: "removed"}
			case m.Status == statusAbandoned:
				out[id] = source.Standing{State: "abandoned"}
			case m.Status == statusInactive:
				out[id] = source.Standing{State: "inactive"}
			case m.IsAvailable != nil && !*m.IsAvailable:
				out[id] = source.Standing{State: "unavailable"}
			}
		}
	}
	return out, nil
}
