package profile

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const maxGroupName = 60

// Group is a named set of profile entry keys that toggle together.
type Group struct {
	Name string   `json:"name"`
	Keys []string `json:"keys"`
}

func cleanGroupName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("group name is empty"))
	}
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

func (s *Store) unlockedGroupName(game, id, name string) (string, error) {
	if err := s.unlocked(game, id); err != nil {
		return "", err
	}
	return cleanGroupName(name)
}

func (s *Store) mutateGroup(game, id, name string, fn func(p *Profile, idx int) error) (Profile, error) {
	name, err := s.unlockedGroupName(game, id, name)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		idx := p.groupIndex(name)
		if idx < 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		return fn(p, idx)
	})
}

// CreateGroup adds an empty group to the profile.
func (s *Store) CreateGroup(game, id, name string) (Profile, error) {
	name, err := s.unlockedGroupName(game, id, name)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		if p.groupIndex(name) >= 0 {
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q already exists", name))
		}
		p.Groups = append(p.Groups, Group{Name: name, Keys: []string{}})
		return nil
	})
}

// RenameGroup changes a group's name.
func (s *Store) RenameGroup(game, id, name, next string) (Profile, error) {
	next, err := cleanGroupName(next)
	if err != nil {
		return Profile{}, err
	}
	return s.mutateGroup(game, id, name, func(p *Profile, idx int) error {
		if p.groupIndex(next) >= 0 && !strings.EqualFold(p.Groups[idx].Name, next) {
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("group %q already exists", next))
		}
		p.Groups[idx].Name = next
		return nil
	})
}

// DeleteGroup removes a group. Entries stay in the profile.
func (s *Store) DeleteGroup(game, id, name string) (Profile, error) {
	return s.mutateGroup(game, id, name, func(p *Profile, idx int) error {
		p.Groups = slices.Delete(p.Groups, idx, idx+1)
		return nil
	})
}

// AddToGroup records an entry key in the named group, creating the group if needed.
func (s *Store) AddToGroup(game, id, name, key string) (Profile, error) {
	name, err := s.unlockedGroupName(game, id, name)
	if err != nil {
		return Profile{}, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("entry key is empty"))
	}
	return s.update(game, id, func(p *Profile, _ string) error {
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

// SetGroupEnabled switches every mod of every entry in the group in one history event.
func (s *Store) SetGroupEnabled(game, id, name string, on bool) (Profile, error) {
	name, err := s.unlockedGroupName(game, id, name)
	if err != nil {
		return Profile{}, err
	}
	kind, change := historyEnabled, ChangeEnabled
	if !on {
		kind, change = historyDisabled, ChangeDisabled
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.updateLockedAs(game, id, kind, HistoryEvent{Change: change, Name: name}, func(p *Profile, dir string) error {
		idx := p.groupIndex(name)
		if idx < 0 {
			return usererr.Wrap(usererr.NotFound, fmt.Errorf("group %q not found", name))
		}
		for _, key := range p.Groups[idx].Keys {
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
		return nil
	})
}
