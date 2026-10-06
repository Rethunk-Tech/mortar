package profile

import (
	"cmp"
	"slices"
	"strings"
)

func lacksLook(e Entry) bool {
	return e.Source.Kind == KindThunderstore && (e.Source.Picture == "" || e.Source.Category == "")
}

// FillPackageLooks records the icon and category of the game's Thunderstore package entries that lack them, from
// the cached index, so a package added before either was recorded shows both. A profile it changes gets no history
// event, since nothing the player chose changed.
func (s *Store) FillPackageLooks(game string) error {
	if s.PackageLooks == nil {
		return nil
	}
	all, err := s.listOK(game)
	if err != nil {
		return err
	}
	var names []string
	for _, p := range all {
		for _, e := range p.Entries {
			if lacksLook(e) {
				names = append(names, e.Source.Name)
			}
		}
	}
	if len(names) == 0 {
		return nil
	}
	looks := s.PackageLooks(game, names)
	if len(looks) == 0 {
		return nil
	}
	for _, p := range all {
		if !slices.ContainsFunc(p.Entries, func(e Entry) bool {
			_, ok := looks[strings.ToLower(e.Source.Name)]
			return ok && lacksLook(e)
		}) {
			continue
		}
		s.setHistoryQuiet(p.ID, true)
		_, err := s.update(game, p.ID, func(p *Profile, _ string) error {
			for i := range p.Entries {
				src := &p.Entries[i].Source
				if l, ok := looks[strings.ToLower(src.Name)]; ok && lacksLook(p.Entries[i]) {
					src.Picture, src.Category = cmp.Or(src.Picture, l.Icon), cmp.Or(src.Category, l.Category)
				}
			}
			return nil
		})
		s.setHistoryQuiet(p.ID, false)
		if err != nil {
			return err
		}
	}
	return nil
}
