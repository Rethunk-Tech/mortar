package modrinth

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/source"
)

// standingBatch is how many projects one request names.
const standingBatch = 100

// Standings flags the projects Modrinth marks archived, one request per hundred projects. An id may be a project id or
// its slug, and the answer is keyed by the id as given.
func (d Driver) Standings(ctx context.Context, version string, ids []string) (map[string]source.Standing, error) {
	out := map[string]source.Standing{}
	for start := 0; start < len(ids); start += standingBatch {
		chunk := ids[start:min(start+standingBatch, len(ids))]
		list, err := json.Marshal(chunk)
		if err != nil {
			return nil, err
		}
		var projects []struct {
			ID     string `json:"id"`
			Slug   string `json:"slug"`
			Status string `json:"status"`
		}
		if err := d.get(ctx, "/projects", url.Values{"ids": {string(list)}}, version, &projects); err != nil {
			return nil, err
		}
		for _, p := range projects {
			if p.Status != "archived" {
				continue
			}
			for _, id := range chunk {
				if strings.EqualFold(id, p.ID) || strings.EqualFold(id, p.Slug) {
					out[id] = source.Standing{State: "archived"}
				}
			}
		}
	}
	return out, nil
}
