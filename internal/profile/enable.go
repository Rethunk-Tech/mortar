package profile

import (
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func requiredNeeds(m Component) []mod.ID {
	opt := make(map[string]struct{}, len(m.Optional))
	for _, id := range m.Optional {
		opt[id.Fold()] = struct{}{}
	}
	var out []mod.ID
	for _, id := range m.Needs {
		if _, skip := opt[id.Fold()]; skip {
			continue
		}
		out = append(out, id)
	}
	return out
}

func modByID(p *Profile, uniqueID mod.ID) (key string, m Component, ok bool) {
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		for _, em := range e.Mods {
			if mod.Equal(em.ID, uniqueID) {
				return e.Key, em, true
			}
		}
	}
	return "", Component{}, false
}

func disabledUID(p *Profile, uniqueID mod.ID) bool {
	for _, e := range p.Entries {
		if hasID(e.Disabled, uniqueID) {
			return true
		}
	}
	return false
}

func (s *Store) autoEnableRequirements(game string) bool {
	if s.settings == nil {
		return true
	}
	return s.settings.Get().GamePrefs(game).AutoEnableRequirements()
}

// enableRequired turns on required dependencies of uniqueID that are already in the profile but switched off.
// Optional dependencies are left as they are. Returns the names that were switched on.
func enableRequired(p *Profile, dir string, uniqueID mod.ID) []string {
	_, self, ok := modByID(p, uniqueID)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{uniqueID.Fold(): {}}
	var also []string
	queue := requiredNeeds(self)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		low := id.Fold()
		if _, done := seen[low]; done {
			continue
		}
		seen[low] = struct{}{}
		key, dep, found := modByID(p, id)
		if !found || !disabledUID(p, dep.ID) {
			continue
		}
		if err := applyEnabled(p, dir, key, dep.ID, true); err != nil {
			continue
		}
		name := dep.Name
		if name == "" {
			name = dep.ID.Local()
		}
		also = append(also, name)
		queue = append(queue, requiredNeeds(dep)...)
	}
	return also
}
