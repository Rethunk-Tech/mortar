package profile

import (
	"strings"

	"github.com/Rethunk-AI/mortar/internal/settings"
)

func requiredNeeds(m EntryMod) []string {
	opt := make(map[string]struct{}, len(m.Optional))
	for _, id := range m.Optional {
		opt[strings.ToLower(id)] = struct{}{}
	}
	var out []string
	for _, id := range m.Needs {
		if _, skip := opt[strings.ToLower(id)]; skip {
			continue
		}
		out = append(out, id)
	}
	return out
}

func modByID(p *Profile, uniqueID string) (key string, m EntryMod, ok bool) {
	for _, e := range p.Entries {
		if isBundled(e) {
			continue
		}
		for _, em := range e.Mods {
			if sameID(em.UniqueID, uniqueID) {
				return e.Key, em, true
			}
		}
	}
	return "", EntryMod{}, false
}

func disabledUID(p *Profile, uniqueID string) bool {
	for _, e := range p.Entries {
		if hasID(e.Disabled, uniqueID) {
			return true
		}
	}
	return false
}

func (s *Store) autoEnableRequirements() bool {
	if s.settings == nil {
		return true
	}
	return s.settings.Get().GamePrefs(settings.GameStardew).AutoEnableRequirements()
}

// enableRequired turns on required dependencies of uniqueID that are already in the profile but switched off.
// Optional dependencies are left as they are. Returns the names that were switched on.
func enableRequired(p *Profile, dir, uniqueID string) []string {
	_, self, ok := modByID(p, uniqueID)
	if !ok {
		return nil
	}
	seen := map[string]struct{}{strings.ToLower(uniqueID): {}}
	var also []string
	queue := requiredNeeds(self)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		low := strings.ToLower(id)
		if _, done := seen[low]; done {
			continue
		}
		seen[low] = struct{}{}
		key, dep, found := modByID(p, id)
		if !found || !disabledUID(p, dep.UniqueID) {
			continue
		}
		if err := applyEnabled(p, dir, key, dep.UniqueID, true); err != nil {
			continue
		}
		name := dep.Name
		if name == "" {
			name = dep.UniqueID
		}
		also = append(also, name)
		queue = append(queue, requiredNeeds(dep)...)
	}
	return also
}
