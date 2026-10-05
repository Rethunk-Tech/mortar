package profile

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// SetWinner records or clears a LoadAfter on winnerKey for loser and rewrites the winner's
// installed manifest so SMAPI loads the winner later.
func (s *Store) SetWinner(game, profileID, winnerKey string, loser mod.ID, on bool) (Profile, error) {
	if loser == "" {
		return Profile{}, fmt.Errorf("loser unique ID is empty")
	}
	var drop []mod.ID
	if !on {
		drop = []mod.ID{loser}
	}
	return s.updateMods(game, profileID, func(p *Profile, dir string) error {
		i := entryIndex(p.Entries, winnerKey)
		if i < 0 {
			return fmt.Errorf("mod %q is not in this profile", winnerKey)
		}
		e := p.Entries[i]
		e.LoadAfter = setLoadAfter(e.LoadAfter, loser, on)
		p.Entries[i] = e
		return applyLoadAfter(filepath.Join(dir, "mods", e.Key), e, drop)
	})
}

// SetWinner records or clears a LoadAfter on winnerKey for loser.
func (s *Service) SetWinner(game, profileID, winnerKey string, loser mod.ID, on bool) (Profile, error) {
	return s.store.SetWinner(game, profileID, winnerKey, loser, on)
}

func entryIndex(entries []Entry, key string) int {
	return slices.IndexFunc(entries, func(e Entry) bool { return e.Key == key })
}

func requireEntry(entries []Entry, key string) (int, error) {
	i := entryIndex(entries, key)
	if i < 0 {
		return 0, fmt.Errorf("%q is not in this profile", key)
	}
	return i, nil
}

func setLoadAfter(ids []mod.ID, loser mod.ID, on bool) []mod.ID {
	out := make([]mod.ID, 0, len(ids)+1)
	for _, id := range ids {
		if !mod.Equal(id, loser) {
			out = append(out, id)
		}
	}
	if on {
		out = append(out, loser)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func applyLoadAfter(root string, e Entry, drop []mod.ID) error {
	if len(e.LoadAfter) == 0 && len(drop) == 0 {
		return nil
	}
	for _, m := range e.Mods {
		rel := filepath.FromSlash(m.Folder)
		if m.Folder == "." {
			rel = ""
		}
		path := filepath.Join(root, rel, manifest.FileName)
		raw, err := fsx.ReadFile(path)
		if err != nil {
			return err
		}
		rewritten, err := rewriteManifestDeps(raw, e.LoadAfter, drop)
		if err != nil {
			return err
		}
		if err := fsx.WriteFile(path, rewritten, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func rewriteManifestDeps(raw []byte, want, drop []mod.ID) ([]byte, error) {
	if _, err := manifest.Parse(raw); err != nil {
		return nil, err
	}
	doc, err := lenientObject(raw)
	if err != nil {
		return nil, err
	}
	type dep struct {
		ID             string `json:"UniqueID"`
		IsRequired     *bool  `json:"IsRequired,omitempty"`
		MinimumVersion string `json:"MinimumVersion,omitempty"`
	}
	var deps []dep
	if v, ok := doc["Dependencies"]; ok {
		_ = json.Unmarshal(v, &deps)
	}
	dropSet, wantSet := idSet(drop), idSet(want)
	kept := make([]dep, 0, len(deps)+len(want))
	seen := map[string]bool{}
	for _, d := range deps {
		low := mod.SMAPI(d.ID).Fold()
		required := d.IsRequired == nil || *d.IsRequired
		if !required && dropSet[low] && !wantSet[low] {
			continue
		}
		kept = append(kept, d)
		seen[low] = true
	}
	off := false
	for _, id := range want {
		if seen[id.Fold()] {
			continue
		}
		kept = append(kept, dep{ID: id.Local(), IsRequired: &off})
		seen[id.Fold()] = true
	}
	if len(kept) == 0 {
		delete(doc, "Dependencies")
	} else {
		b, err := json.Marshal(kept)
		if err != nil {
			return nil, err
		}
		doc["Dependencies"] = b
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func idSet(ids []mod.ID) map[string]bool {
	out := map[string]bool{}
	for _, id := range ids {
		out[id.Fold()] = true
	}
	return out
}

func lenientObject(raw []byte) (map[string]json.RawMessage, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(jsonc.Clean(raw), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}
