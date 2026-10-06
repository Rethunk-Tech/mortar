package control

import (
	"context"
	"fmt"

	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// externalSources lists the other mod managers' profiles for the game, of one kind (Source) or all.
func (s *Services) externalSources(p Params) ([]migrate.SourceInfo, error) {
	if s.Profiles == nil {
		return nil, errUnavailable
	}
	all, err := s.Profiles.ExternalSources(p.Game)
	if err != nil {
		return nil, err
	}
	out := []migrate.SourceInfo{}
	for _, src := range all {
		if p.Source == "" || src.Kind == p.Source {
			out = append(out, src)
		}
	}
	return out, nil
}

// externalImport imports the Source manager's profile ID (its id or name) as the import wizard does by default, or
// with Preview returns what it would import.
func (s *Services) externalImport(ctx context.Context, p Params) (any, error) {
	if s.Profiles == nil {
		return nil, errUnavailable
	}
	if p.Source == "" {
		return nil, usererr.New(usererr.Invalid, "name the mod manager to import from")
	}
	sources, err := s.externalSources(p)
	if err != nil {
		return nil, err
	}
	id := ""
	for _, src := range sources {
		for _, prof := range src.Profiles {
			if prof.ID == p.ID || (id == "" && prof.Name == p.ID) {
				id = prof.ID
			}
		}
	}
	if id == "" {
		return nil, usererr.Wrap(usererr.NotFound, fmt.Errorf("no %s profile %q for %s", p.Source, p.ID, p.Game))
	}
	pv, err := s.Profiles.ExternalPreview(p.Game, p.Source, id)
	if err != nil || p.Preview {
		return pv, err
	}
	if s.Shares == nil {
		return nil, errUnavailable
	}
	return s.Shares.ImportExternal(ctx, p.Game, pv)
}
