package profile

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

const latestStoreKey = "latest"

const (
	skipPinned  = "pinned"
	skipVersion = "skip-version"
	skipLocked  = "locked"
	skipSameKey = "already at this version"
	skipChoices = "needs install choices"
)

// EverywhereHit is a profile that can take the same store-key update.
type EverywhereHit struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	OldKey    string `json:"oldKey"`
	// Change is the history event the update recorded in this profile.
	Change string `json:"change,omitempty"`
}

// EverywhereSkip is a profile that holds the mod but will not be updated.
type EverywhereSkip struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	Reason    string `json:"reason"`
}

// EverywherePreview lists who would receive UpdateEverywhere and who is left out.
type EverywherePreview struct {
	Affected []EverywhereHit  `json:"affected"`
	Skipped  []EverywhereSkip `json:"skipped"`
}

// EverywhereResult is what UpdateEverywhere changed and skipped.
type EverywhereResult struct {
	Updated []EverywhereHit  `json:"updated"`
	Skipped []EverywhereSkip `json:"skipped"`
}

// PreviewEverywhere reports which profiles hold modKeyOrID (entry key, mod id, or the same Nexus mod)
// and which are excluded because they are pinned, skip-version, or locked (game running).
func (s *Store) PreviewEverywhere(game, modKeyOrID string) (EverywherePreview, error) {
	all, err := s.listOK(game)
	if err != nil {
		return EverywherePreview{}, err
	}
	id := identityOf(all, modKeyOrID)
	var out EverywherePreview
	for _, p := range all {
		ei := id.match(p)
		if ei < 0 {
			continue
		}
		e := p.Entries[ei]
		if err := s.unlocked(game, p.ID); err != nil {
			out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: p.ID, Name: p.Name, Reason: skipLocked})
			continue
		}
		if e.Pinned {
			out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: p.ID, Name: p.Name, Reason: skipPinned})
			continue
		}
		if strings.TrimSpace(e.SkipVersion) != "" {
			out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: p.ID, Name: p.Name, Reason: skipVersion})
			continue
		}
		out.Affected = append(out.Affected, EverywhereHit{ProfileID: p.ID, Name: p.Name, OldKey: e.Key})
	}
	return out, nil
}

type everywhereID struct {
	query  string
	ids    map[string]struct{}
	modIDs map[int]struct{}
	keys   map[string]struct{}
}

func identityOf(all []Profile, q string) everywhereID {
	id := everywhereID{
		query:  strings.TrimSpace(q),
		ids:    map[string]struct{}{},
		modIDs: map[int]struct{}{},
		keys:   map[string]struct{}{},
	}
	if id.query == "" {
		return id
	}
	if modID, _, ok := store.NexusFile(id.query); ok {
		id.modIDs[modID] = struct{}{}
		id.keys[id.query] = struct{}{}
	}
	for _, p := range all {
		for _, e := range p.Entries {
			if e.Key == id.query || id.hasMod(e) {
				id.note(e)
			}
		}
	}
	if len(id.modIDs) == 0 {
		return id
	}
	for _, p := range all {
		for _, e := range p.Entries {
			if _, ok := id.modIDs[e.Source.ModID]; ok && e.Source.ModID != 0 {
				id.note(e)
			}
		}
	}
	return id
}

func (id everywhereID) note(e Entry) {
	id.keys[e.Key] = struct{}{}
	if e.Source.ModID != 0 {
		id.modIDs[e.Source.ModID] = struct{}{}
	}
	for _, m := range e.Mods {
		if m.ID != "" {
			id.ids[m.ID.Fold()] = struct{}{}
		}
	}
}

func (id everywhereID) hasMod(e Entry) bool {
	for _, m := range e.Mods {
		if mod.Equal(m.ID, mod.ID(id.query)) {
			return true
		}
	}
	return false
}

func (id everywhereID) match(p Profile) int {
	if id.query == "" {
		return -1
	}
	for i, e := range p.Entries {
		if _, ok := id.keys[e.Key]; ok {
			return i
		}
		if e.Source.ModID != 0 {
			if _, ok := id.modIDs[e.Source.ModID]; ok {
				return i
			}
		}
		for _, m := range e.Mods {
			if _, ok := id.ids[m.ID.Fold()]; ok {
				return i
			}
		}
	}
	return -1
}

// UpdateEverywhere replaces the matching entry in every eligible profile with newStoreKey
// (empty or "latest" means the newest store item that shares mod id / Nexus identity).
// The store item is used once; each profile records its own history (and so its own Undo).
func (s *Store) UpdateEverywhere(game, modKeyOrID, newStoreKey string) (EverywhereResult, error) {
	preview, err := s.PreviewEverywhere(game, modKeyOrID)
	if err != nil {
		return EverywhereResult{}, err
	}
	out := EverywhereResult{Skipped: append([]EverywhereSkip{}, preview.Skipped...)}
	if len(preview.Affected) == 0 {
		return out, nil
	}
	target := strings.TrimSpace(newStoreKey)
	if target == "" || strings.EqualFold(target, latestStoreKey) {
		target, err = s.latestStoreKey(game, preview.Affected[0].OldKey, modKeyOrID)
		if err != nil {
			return EverywhereResult{}, err
		}
	}
	src := s.SourceOf(game, target)
	var source *Source
	if src.Kind != "" {
		source = &src
	}
	for _, hit := range preview.Affected {
		if err := s.unlocked(game, hit.ProfileID); err != nil {
			out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: hit.ProfileID, Name: hit.Name, Reason: skipLocked})
			continue
		}
		p, err := s.read(game, hit.ProfileID)
		if err != nil {
			return out, err
		}
		all, err := s.listOK(game)
		if err != nil {
			return out, err
		}
		ei := identityOf(all, modKeyOrID).match(p)
		if ei < 0 {
			continue
		}
		oldKey := p.Entries[ei].Key
		if oldKey == target {
			out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: hit.ProfileID, Name: hit.Name, Reason: skipSameKey})
			continue
		}
		var updated Profile
		var applyErr error
		if len(p.Entries[ei].ExtraStoreKeys) > 0 {
			updated, applyErr = s.UpdateMultiFile(game, hit.ProfileID, oldKey, target, source)
		} else {
			updated, applyErr = s.UpdateEntry(game, hit.ProfileID, oldKey, target)
		}
		if applyErr != nil {
			if _, ok := errors.AsType[*NeedChoicesError](applyErr); ok {
				out.Skipped = append(out.Skipped, EverywhereSkip{ProfileID: hit.ProfileID, Name: hit.Name, Reason: skipChoices})
				continue
			}
			return out, applyErr
		}
		out.Updated = append(out.Updated, EverywhereHit{ProfileID: hit.ProfileID, Name: hit.Name, OldKey: oldKey, Change: updated.LastChange})
	}
	return out, nil
}

// latestStoreKey is the store item holding the newest later version of what oldKey holds. A site package
// (Thunderstore, GitHub, Modrinth) is matched by source and package name and ordered by its version; a Nexus file by
// mod page and upload order, keeping to files that hold the same mod; anything else by the mod inside and its
// manifest version. An item whose version cannot be ordered is never picked.
func (s *Store) latestStoreKey(game, oldKey, modKeyOrID string) (string, error) {
	keys, err := s.items.Keys(game)
	if err != nil {
		return "", err
	}
	oldMods := s.scanKey(game, oldKey)
	primary := primaryMod(oldMods, mod.ID(strings.TrimSpace(modKeyOrID)))
	oldSrc, oldPkg, oldVer, _ := s.items.Meta(game, oldKey)
	oldPage, bestFile, nexus := store.NexusFile(oldKey)
	best, bestVer := "", oldVer
	if !nexus && oldPkg == "" {
		bestVer = heldVersion(oldMods, primary)
	}
	for _, key := range keys {
		if key == oldKey {
			continue
		}
		switch {
		case nexus:
			page, file, ok := store.NexusFile(key)
			if !ok || page != oldPage || file <= bestFile {
				continue
			}
			if primary != "" && heldVersion(s.scanKey(game, key), primary) == "" {
				continue
			}
			best, bestFile = key, file
		case oldPkg != "":
			src, pkg, ver, _ := s.items.Meta(game, key)
			if src == oldSrc && strings.EqualFold(pkg, oldPkg) && meta.Newer(ver, bestVer) {
				best, bestVer = key, ver
			}
		case primary != "":
			if ver := heldVersion(s.scanKey(game, key), primary); meta.Newer(ver, bestVer) {
				best, bestVer = key, ver
			}
		}
	}
	if best == "" {
		return "", fmt.Errorf("no later version in the store")
	}
	return best, nil
}

func (s *Store) scanKey(game, key string) []manifest.Mod {
	root, err := s.items.Path(game, key)
	if err != nil {
		return nil
	}
	found, _ := manifest.Scan(root)
	return found
}

// primaryMod is the mod whose version orders an item: the one asked for when the item holds it, else its first.
func primaryMod(found []manifest.Mod, want mod.ID) mod.ID {
	for _, m := range found {
		if want != "" && mod.Equal(m.ModID(), want) {
			return m.ModID()
		}
	}
	if len(found) == 0 {
		return ""
	}
	return found[0].ModID()
}

func heldVersion(found []manifest.Mod, id mod.ID) string {
	for _, m := range found {
		if mod.Equal(m.ModID(), id) {
			return m.Version
		}
	}
	return ""
}
