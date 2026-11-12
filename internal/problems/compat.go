package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-semverish"
)

// Compat is one profile mod's SMAPI compatibility-list row: advice, counted until dismissed.
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

func (s *Service) compatIndex(ctx context.Context, gameID string) (smapiweb.CompatIndex, bool) {
	if info, ok := components.Game(gameID); !ok || !info.HasMetadata(components.MetaSMAPICompat) {
		return smapiweb.CompatIndex{}, false
	}
	idx, err := s.meta.CompatList(ctx)
	if err != nil {
		return smapiweb.CompatIndex{}, false
	}
	return idx, true
}

func matchCompat(idx smapiweb.CompatIndex, mods []framework.Mod, skipOK bool) []Compat {
	out := []Compat{}
	seen := map[string]bool{}
	for _, m := range mods {
		// The check is about what runs: a switched-off mod is listed only when asked about on its own.
		if skipOK && !m.Enabled {
			continue
		}
		e, ok := idx.Lookup(m.ModID().Local(), nexusIDOf(m))
		if !ok {
			continue
		}
		if skipOK && e.Status == smapiweb.StatusOK {
			continue
		}
		if e.Status == smapiweb.StatusUnofficial && installedMeetsUnofficial(m.Version, e.UnofficialVersion) {
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

// installedMeetsUnofficial reports whether the installed version is the unofficial update or newer, which is what the row
// would ask for. An unparseable version on either side keeps the row.
func installedMeetsUnofficial(installed, unofficial string) bool {
	if unofficial == "" {
		return false
	}
	c, ok := semverish.Compare(installed, unofficial)
	return ok && c >= 0
}

func nexusIDOf(m framework.Mod) int {
	return smapimanifest.FirstNexusID(m.UpdateKeys)
}

// gatedMeta answers only the lookups the game's catalog entry names; a game that names none asks nothing.
type gatedMeta struct {
	Meta
	info components.GameInfo
}

func (s *Service) metaFor(gameID string) gatedMeta {
	info, _ := components.Game(gameID)
	return gatedMeta{Meta: s.meta, info: info}
}

func (m gatedMeta) CheckUpdates(ctx context.Context, req meta.UpdateRequest) []smapiweb.UpdateResult {
	if !m.info.HasMetadata(components.MetaSMAPIUpdates) {
		return nil
	}
	return m.Meta.CheckUpdates(ctx, req)
}

func (m gatedMeta) Lookup(ctx context.Context, uniqueID string) ([]smapiweb.Ref, error) {
	if !m.info.HasMetadata(components.MetaStardewDataset) {
		return nil, nil
	}
	return m.Meta.Lookup(ctx, uniqueID)
}
