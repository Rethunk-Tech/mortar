package problems

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/loader/smapi/smapimanifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexussvc"
	"github.com/Rethunk-Tech/mortar/internal/profile"

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

// ForGame is m answering only the lookups gameID's catalog entry names.
func ForGame(gameID string, m Meta) Meta {
	info, _ := components.Game(gameID)
	return gatedMeta{Meta: m, info: info}
}

func (m gatedMeta) CheckUpdates(ctx context.Context, req meta.UpdateRequest) []smapiweb.UpdateResult {
	if !m.info.HasMetadata(components.MetaSMAPIUpdates) {
		return nil
	}
	return m.Meta.CheckUpdates(ctx, req)
}

// Page answers an empty page outside Stardew: the dataset is keyed by Stardew's Nexus ids, which another game's ids
// collide with.
func (m gatedMeta) Page(ctx context.Context, id int) (smapiweb.Page, error) {
	if !m.info.HasMetadata(components.MetaStardewDataset) {
		return smapiweb.Page{}, nil
	}
	return m.Meta.Page(ctx, id)
}

func (m gatedMeta) Lookup(ctx context.Context, uniqueID string) ([]smapiweb.Ref, error) {
	if !m.info.HasMetadata(components.MetaStardewDataset) {
		return nil, nil
	}
	return m.Meta.Lookup(ctx, uniqueID)
}

// renameDatasetNamed renames the Nexus entries of a game without the Stardew dataset that were saved under the
// dataset's name for their mod id, which another game's ids collide with, to their own page's name.
func (s *Service) renameDatasetNamed(ctx context.Context, gameID, id, domain string) {
	if info, _ := components.Game(gameID); domain == "" || s.meta == nil || info.HasMetadata(components.MetaStardewDataset) {
		return
	}
	p, err := s.profiles.Get(gameID, id)
	if err != nil {
		return
	}
	names := datasetNamed(ctx, s.meta, s.ownPageNames(ctx, gameID, domain, p.Entries), p.Entries)
	if len(names) == 0 {
		return
	}
	if _, err := s.profiles.RenameEntries(gameID, id, names); err != nil {
		slog.Warn("problems: rename entries named from the Stardew dataset", "game", gameID, "err", err)
	}
}

// ownPageNames is the name of each Nexus entry's page in the game's own domain: the batched lookup, else the cache.
func (s *Service) ownPageNames(ctx context.Context, gameID, domain string, entries []profile.Entry) map[int]string {
	var ids []int
	for _, e := range entries {
		if e.Source.Kind == profile.KindNexus && e.Source.ModID > 0 && e.Source.ModName != "" {
			ids = append(ids, e.Source.ModID)
		}
	}
	out := map[int]string{}
	if len(ids) == 0 {
		return out
	}
	if s.NexusPages != nil {
		// A lookup cut short still returns the pages it got; the cache fills the rest.
		pages, _ := s.NexusPages(ctx, gameID, ids)
		for id, page := range pages {
			out[id] = page.Name
		}
	}
	cl := &meta.Client{CacheDir: filepath.Join(s.home, "cache")}
	for _, id := range ids {
		if out[id] != "" {
			continue
		}
		if d, ok := nexussvc.PeekDetails(cl, domain, id); ok {
			out[id] = d.Page.Name
		}
	}
	return out
}

// datasetNamed maps each Nexus entry whose name is not its own page's but the Stardew dataset's for the same mod id
// to its own page's name. The dataset is asked only about those mismatches.
func datasetNamed(ctx context.Context, dataset Meta, own map[int]string, entries []profile.Entry) map[string]string {
	out := map[string]string{}
	for _, e := range entries {
		src := e.Source
		name := own[src.ModID]
		if src.Kind != profile.KindNexus || src.ModID <= 0 || src.ModName == "" || name == "" || name == src.ModName {
			continue
		}
		if wrong, err := dataset.Page(ctx, src.ModID); err == nil && wrong.Name == src.ModName {
			out[e.Key] = name
		}
	}
	return out
}
