package profile

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const maxGroupName = 60

// GroupSep splits a group name into the path of nested groups: "Visuals/Hats" is the group Hats inside Visuals. The
// name stays one flat string, so a Mortar that does not nest reads the same profile or share as a plain group of that
// name. A parent needs no group of its own; it exists while any name has it as a prefix.
const GroupSep = "/"

// InGroup reports whether name is group or lies inside it.
func InGroup(name, group string) bool {
	return strings.EqualFold(name, group) || strings.HasPrefix(strings.ToLower(name), strings.ToLower(group)+GroupSep)
}

// Group is a named set of profile entry keys that toggle together.
type Group struct {
	Name string   `json:"name"`
	Keys []string `json:"keys"`
}

// CleanGroupName trims a group name and each level of its path, and refuses an empty name or level.
func CleanGroupName(name string) (string, error) {
	parts := strings.Split(name, GroupSep)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
		if parts[i] == "" && (len(parts) > 1 || strings.TrimSpace(name) == "") {
			return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("group name %q has an empty level", name))
		}
	}
	name = strings.Join(parts, GroupSep)
	if utf8.RuneCountInString(name) > maxGroupName {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("group name is longer than %d characters", maxGroupName))
	}
	return name, nil
}

func (p Profile) groupIndex(name string) int {
	for i, g := range p.Groups {
		if strings.EqualFold(g.Name, name) {
			return i
		}
	}
	return -1
}

func dropKeyFromGroups(p *Profile, key string) {
	out := p.Groups[:0]
	for _, g := range p.Groups {
		g.Keys = slices.DeleteFunc(g.Keys, func(k string) bool { return k == key })
		out = append(out, g)
	}
	p.Groups = out
}

func (s *Store) mutateGroup(game, id, name string, fn func(p *Profile, idx int) error) (Profile, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		idx := p.groupIndex(name)
		if idx < 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		return fn(p, idx)
	})
}

// CreateGroup adds an empty group to the profile.
func (s *Store) CreateGroup(game, id, name string) (Profile, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		if p.groupIndex(name) >= 0 {
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q already exists", name))
		}
		p.Groups = append(p.Groups, Group{Name: name, Keys: []string{}})
		return nil
	})
}

// freeGroupName is name, or when the profile has a group by that name the first of "name (2)", "name (3)" and so on
// that it does not, cut to fit the length limit.
func (p Profile) freeGroupName(name string) string {
	if p.groupIndex(name) < 0 {
		return name
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprintf(" (%d)", n)
		base := []rune(name)
		if over := len(base) + utf8.RuneCountInString(suffix) - maxGroupName; over > 0 {
			base = base[:len(base)-over]
		}
		if candidate := string(base) + suffix; p.groupIndex(candidate) < 0 {
			return candidate
		}
	}
}

// AddGroup creates a group holding those of keys that name entries of the profile, under the first free name from
// freeGroupName, and returns the profile and the name it took.
func (s *Store) AddGroup(game, id, name string, keys []string) (Profile, string, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, "", err
	}
	var used string
	p, err := s.updateMods(game, id, func(p *Profile, _ string) error {
		used = p.freeGroupName(name)
		held := make([]string, 0, len(keys))
		for _, key := range keys {
			if entryIndex(p.Entries, key) >= 0 && !slices.Contains(held, key) {
				held = append(held, key)
			}
		}
		p.Groups = append(p.Groups, Group{Name: used, Keys: held})
		return nil
	})
	return p, used, err
}

// groupsInside are the indexes of the groups named group or lying inside it.
func (p Profile) groupsInside(group string) []int {
	var out []int
	for i, g := range p.Groups {
		if InGroup(g.Name, group) {
			out = append(out, i)
		}
	}
	return out
}

// RenameGroup changes a group's name, and the path prefix of every group inside it, so renaming moves the whole tree.
func (s *Store) RenameGroup(game, id, name, next string) (Profile, error) {
	next, err := CleanGroupName(next)
	if err != nil {
		return Profile{}, err
	}
	name, err = CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		inside := p.groupsInside(name)
		if len(inside) == 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		if InGroup(next, name) && !strings.EqualFold(next, name) {
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q cannot move inside itself", name))
		}
		renamed := make(map[int]string, len(inside))
		for _, i := range inside {
			to := next + string([]rune(p.Groups[i].Name)[utf8.RuneCountInString(name):])
			if utf8.RuneCountInString(to) > maxGroupName {
				return usererr.Wrap(usererr.Invalid, fmt.Errorf("group name %q is longer than %d characters", to, maxGroupName))
			}
			if j := p.groupIndex(to); j >= 0 && !slices.Contains(inside, j) {
				return usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q already exists", to))
			}
			renamed[i] = to
		}
		for i, to := range renamed {
			p.Groups[i].Name = to
		}
		return nil
	})
}

// DeleteGroup removes a group and the groups inside it. Entries stay in the profile.
func (s *Store) DeleteGroup(game, id, name string) (Profile, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		inside := p.groupsInside(name)
		if len(inside) == 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		p.Groups = slices.DeleteFunc(p.Groups, func(g Group) bool { return InGroup(g.Name, name) })
		return nil
	})
}

// AddToGroup records an entry key in the named group, creating the group if needed.
func (s *Store) AddToGroup(game, id, name, key string) (Profile, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("entry key is empty"))
	}
	return s.updateMods(game, id, func(p *Profile, _ string) error {
		if entryIndex(p.Entries, key) < 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("entry %q not found", key))
		}
		idx := p.groupIndex(name)
		if idx < 0 {
			p.Groups = append(p.Groups, Group{Name: name, Keys: []string{key}})
			return nil
		}
		if slices.Contains(p.Groups[idx].Keys, key) {
			return nil
		}
		p.Groups[idx].Keys = append(p.Groups[idx].Keys, key)
		return nil
	})
}

// RemoveFromGroup drops an entry key from the named group.
func (s *Store) RemoveFromGroup(game, id, name, key string) (Profile, error) {
	key = strings.TrimSpace(key)
	return s.mutateGroup(game, id, name, func(p *Profile, idx int) error {
		p.Groups[idx].Keys = slices.DeleteFunc(p.Groups[idx].Keys, func(k string) bool { return k == key })
		return nil
	})
}

// SetGroupEnabled switches every mod of every entry in the group, and in the groups inside it, in one history event.
func (s *Store) SetGroupEnabled(game, id, name string, on bool) (Profile, error) {
	name, err := CleanGroupName(name)
	if err != nil {
		return Profile{}, err
	}
	kind, change := historyEnabled, ChangeEnabled
	if !on {
		kind, change = historyDisabled, ChangeDisabled
	}
	if err := s.lockUnlocked(game, id); err != nil {
		return Profile{}, err
	}
	defer s.mu.Unlock()
	return s.updateLockedAs(game, id, kind, HistoryEvent{Change: change, Name: name}, func(p *Profile, dir string) error {
		inside := p.groupsInside(name)
		if len(inside) == 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		for _, gi := range inside {
			for _, key := range p.Groups[gi].Keys {
				ei := entryIndex(p.Entries, key)
				if ei < 0 {
					continue
				}
				e := p.Entries[ei]
				if e.Source.Bundled() {
					continue
				}
				for _, m := range e.Mods {
					if err := applyEnabled(p, dir, key, m.ID, on); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

// renameGroupKey keeps an entry in its groups when its key changes, as an update or a roll back does.
func renameGroupKey(groups []Group, old, renamed string) {
	for i := range groups {
		for j, k := range groups[i].Keys {
			if k == old {
				groups[i].Keys[j] = renamed
			}
		}
	}
}
