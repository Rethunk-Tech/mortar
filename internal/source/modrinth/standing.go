package modrinth

import (
	"context"
	"strings"

	gomodrinth "github.com/Rethunk-Tech/go-modrinth"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// Standings flags the projects Modrinth marks archived, one request per hundred projects. An id may be a project id or
// its slug, and the answer is keyed by the id as given.
func (d Driver) Standings(ctx context.Context, version string, ids []string) (map[string]source.Standing, error) {
	out := map[string]source.Standing{}
	for start := 0; start < len(ids); start += gomodrinth.ProjectsBatch {
		chunk := ids[start:min(start+gomodrinth.ProjectsBatch, len(ids))]
		projects, err := d.projects(ctx, chunk, version)
		if err != nil {
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

func (d Driver) projects(ctx context.Context, ids []string, version string) ([]gomodrinth.Project, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	projects, err := d.api(version).Projects(ctx, ids)
	return projects, apiErr(err)
}
