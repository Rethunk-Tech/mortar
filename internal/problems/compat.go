package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/metadata"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Compat is one profile mod's SMAPI compatibility-list row. It is informational and is not counted.
type Compat struct {
	Key           string `json:"key"`
	ID            mod.ID `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Summary       string `json:"summary"`
	BrokeIn       string `json:"brokeIn"`
	UnofficialURL string `json:"unofficialUrl"`
	Replacement   string `json:"replacement"`
}

// CompatibilityFor returns SMAPI compatibility-list rows for the profile's mods (including status ok).
//
//wails:ignore
func (s *Service) CompatibilityFor(ctx context.Context, gameID, id string) ([]Compat, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	idx, ok := s.compatIndex(ctx, gameID)
	if !ok {
		return []Compat{}, nil
	}
	return matchCompat(idx, mods, false), nil
}

func (s *Service) withCompat(ctx context.Context, r Result, gameID string, mods []framework.Mod, sameJob []framework.Redundant) Result {
	idx, ok := s.compatIndex(ctx, gameID)
	switch {
	case ok:
		r.Compat = matchCompat(idx, mods, true)
	case r.Compat == nil:
		r.Compat = []Compat{}
	}
	return withSameJob(superseded(r, nexusDomain(gameID), mods), sameJob)
}

func (s *Service) compatIndex(ctx context.Context, gameID string) (meta.CompatIndex, bool) {
	st := s.providers(gameID).Status
	if st == nil {
		return meta.CompatIndex{}, false
	}
	idx, err := st.CompatList(ctx)
	if err != nil {
		return meta.CompatIndex{}, false
	}
	return idx, true
}

func matchCompat(idx meta.CompatIndex, mods []framework.Mod, skipOK bool) []Compat {
	out := []Compat{}
	seen := map[string]bool{}
	for _, m := range mods {
		e, ok := idx.Lookup(m.ModID().Local(), nexusIDOf(m))
		if !ok {
			continue
		}
		if skipOK && e.Status == meta.StatusOK {
			continue
		}
		key := strings.ToLower(m.Key) + "\x00" + m.ModID().Fold()
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Compat{
			Key: m.Key, ID: m.ModID(), Name: m.Name,
			Status: e.Status, Summary: e.Summary, BrokeIn: e.BrokeIn,
			UnofficialURL: e.UnofficialURL, Replacement: e.Replacement,
		})
	}
	return out
}

func nexusIDOf(m framework.Mod) int {
	for _, key := range m.UpdateKeys {
		if n, ok := manifest.NexusUpdateKey(key); ok {
			return n
		}
	}
	return 0
}

// providers are the metadata providers the game's catalog entry names; a game with no client has none.
func (s *Service) providers(gameID string) metadata.Providers {
	c, ok := s.meta.(*meta.Client)
	if !ok {
		return metadata.Providers{}
	}
	for _, g := range game.Catalog() {
		if g.ID == gameID {
			return metadata.For(c, g.Metadata)
		}
	}
	return metadata.Providers{}
}

// metaFor is the lookup surface for a game's checks: updates and the dataset answer from the game's providers,
// and a game without one asks nothing. A Meta that is not the shared client (a test fake) answers as it is.
func (s *Service) metaFor(gameID string) providerMeta {
	if _, ok := s.meta.(*meta.Client); !ok {
		return providerMeta{Meta: s.meta, asIs: true}
	}
	return providerMeta{Meta: s.meta, p: s.providers(gameID)}
}

type providerMeta struct {
	Meta
	p    metadata.Providers
	asIs bool
}

func (m providerMeta) CheckUpdates(ctx context.Context, req meta.UpdateRequest) []meta.UpdateResult {
	switch {
	case m.asIs:
		return m.Meta.CheckUpdates(ctx, req)
	case m.p.Updates == nil:
		return nil
	}
	return m.p.Updates.CheckUpdates(ctx, req)
}

func (m providerMeta) Lookup(ctx context.Context, uniqueID string) ([]meta.Ref, error) {
	switch {
	case m.asIs:
		return m.Meta.Lookup(ctx, uniqueID)
	case m.p.Dataset == nil:
		return nil, nil
	}
	return m.p.Dataset.Lookup(ctx, uniqueID)
}
