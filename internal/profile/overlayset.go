package profile

import (
	"path"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// OverlayFileSet is what one optional file does to its main file's folder: the paths it replaces (the main file
// has them) and the paths it adds, both inside the main entry's folder. Alternatives are the other optional files
// of the same main file that replace any of the same paths; only one of them is on at a time.
type OverlayFileSet struct {
	Key          string   `json:"key"`
	Replaces     []string `json:"replaces"`
	Adds         []string `json:"adds"`
	Alternatives []string `json:"alternatives"`
}

// WithReplacing returns a copy that, installed as an optional file, takes the place of the optional file of the same
// mod at Nexus file fileID: a newer version of it keeps its slot, placement and switch.
func (s Source) WithReplacing(fileID int) Source {
	s.replacing = fileID
	return s
}

// overlayFileSets works out an OverlayFileSet for every optional file laid over baseKey, in profile order.
func (s *Store) overlayFileSets(game, id string, entries []Entry, baseKey string) ([]OverlayFileSet, error) {
	bi := slices.IndexFunc(entries, func(e Entry) bool { return e.Key == baseKey && !e.IsOverlay() })
	overs := overlaysOf(entries, baseKey)
	if bi < 0 || len(overs) == 0 {
		return []OverlayFileSet{}, nil
	}
	baseSrc, tmp, err := s.layoutItem(game, id, baseKey, entries[bi].Fomod)
	if tmp != "" {
		defer func() { _ = fsx.RemoveAll(tmp) }()
	}
	if err != nil {
		return nil, err
	}
	baseFiles, err := overlayFiles(baseSrc)
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(baseFiles))
	for _, f := range baseFiles {
		have[strings.ToLower(f)] = true
	}
	out := make([]OverlayFileSet, 0, len(overs))
	for _, o := range overs {
		_, files, err := s.overlaySource(game, o)
		if err != nil {
			return nil, err
		}
		to, err := overlayRel(o.OverlayTo)
		if err != nil {
			return nil, err
		}
		set := OverlayFileSet{Key: o.Key, Replaces: []string{}, Adds: []string{}, Alternatives: []string{}}
		for _, f := range files {
			rel := path.Join(to, f)
			if have[strings.ToLower(rel)] {
				set.Replaces = append(set.Replaces, rel)
			} else {
				set.Adds = append(set.Adds, rel)
			}
		}
		out = append(out, set)
	}
	for i := range out {
		for j := range out {
			if i != j && sharesPath(out[i].Replaces, out[j].Replaces) {
				out[i].Alternatives = append(out[i].Alternatives, out[j].Key)
			}
		}
	}
	return out, nil
}

func sharesPath(a, b []string) bool {
	seen := make(map[string]bool, len(a))
	for _, p := range a {
		seen[strings.ToLower(p)] = true
	}
	return slices.ContainsFunc(b, func(p string) bool { return seen[strings.ToLower(p)] })
}

// offAlternatives switches off every optional file that is an alternative of key, so key is the one in use.
func (s *Store) offAlternatives(game string, p *Profile, key string) error {
	i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key && e.IsOverlay() })
	if i < 0 {
		return nil
	}
	sets, err := s.overlayFileSets(game, p.ID, p.Entries, p.Entries[i].OverlayOf)
	if err != nil {
		return err
	}
	for _, set := range sets {
		if set.Key != key {
			continue
		}
		for j := range p.Entries {
			if slices.Contains(set.Alternatives, p.Entries[j].Key) {
				p.Entries[j].OverlayOff = true
			}
		}
	}
	return nil
}

// OverlayFiles lists what each optional file laid over baseKey replaces and adds, and which are alternatives.
func (s *Store) OverlayFiles(game, id, baseKey string) ([]OverlayFileSet, error) {
	s.mu.Lock()
	p, err := s.read(game, id)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return s.overlayFileSets(game, id, p.Entries, baseKey)
}

func (s *Service) OverlayFiles(game, id, baseKey string) ([]OverlayFileSet, error) {
	return s.store.OverlayFiles(game, id, baseKey)
}

// StoredOverlay reports a Nexus store item that installs as an optional file laid over its mod's main file.
func (s *Store) StoredOverlay(game, key string) bool {
	modID, _, ok := store.NexusFile(key)
	if !ok {
		return false
	}
	over, err := s.isOverlayItem(game, key, Source{Kind: KindNexus, ModID: modID})
	return err == nil && over
}
