package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

// Compat is one profile mod's SMAPI compatibility-list row. It is informational and is not counted.
type Compat struct {
	Key           string `json:"key"`
	UniqueID      string `json:"uniqueId"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	Summary       string `json:"summary"`
	BrokeIn       string `json:"brokeIn"`
	UnofficialURL string `json:"unofficialUrl"`
	Replacement   string `json:"replacement"`
}

type compatLister interface {
	CompatList(context.Context) (meta.CompatIndex, error)
}

// CompatibilityFor returns SMAPI compatibility-list rows for the profile's mods (including status ok).
func (s *Service) CompatibilityFor(ctx context.Context, gameID, id string) ([]Compat, error) {
	if gameID != "" && gameID != "stardew" {
		return []Compat{}, nil
	}
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	idx, ok := s.compatIndex(ctx)
	if !ok {
		return []Compat{}, nil
	}
	return matchCompat(idx, mods, false), nil
}

func (s *Service) withCompat(ctx context.Context, gameID, id string, r Result, mods []Installed) Result {
	idx, ok := s.compatIndex(ctx)
	switch {
	case ok:
		r.Compat = matchCompat(idx, mods, true)
	case r.Compat == nil:
		r.Compat = []Compat{}
	}
	return s.withSameJob(gameID, id, superseded(r, mods), mods)
}

func (s *Service) compatIndex(ctx context.Context) (meta.CompatIndex, bool) {
	c, ok := s.meta.(compatLister)
	if !ok {
		return meta.CompatIndex{}, false
	}
	idx, err := c.CompatList(ctx)
	if err != nil {
		return meta.CompatIndex{}, false
	}
	return idx, true
}

func matchCompat(idx meta.CompatIndex, mods []Installed, skipOK bool) []Compat {
	out := []Compat{}
	seen := map[string]bool{}
	for _, m := range mods {
		e, ok := idx.Lookup(m.UniqueID, nexusIDOf(m))
		if !ok {
			continue
		}
		if skipOK && e.Status == meta.StatusOK {
			continue
		}
		key := strings.ToLower(m.Key) + "\x00" + strings.ToLower(m.UniqueID)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Compat{
			Key: m.Key, UniqueID: m.UniqueID, Name: m.Name,
			Status: e.Status, Summary: e.Summary, BrokeIn: e.BrokeIn,
			UnofficialURL: e.UnofficialURL, Replacement: e.Replacement,
		})
	}
	return out
}

func nexusIDOf(m Installed) int {
	for _, key := range m.UpdateKeys {
		if n, ok := nexusKey(key); ok {
			return n
		}
	}
	return 0
}
