package profile

import (
	"fmt"
	"slices"

	gamereg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// LaunchOptions returns the profile's extra SMAPI arguments as stored.
func (s *Store) LaunchOptions(game, id string) (string, error) {
	p, err := s.read(game, id)
	if err != nil {
		return "", err
	}
	return p.LaunchOptions, nil
}

// MaxTestCopies bounds a multiplayer test on one computer: a host and a few clients.
const MaxTestCopies = 4

// SetTestCopies remembers how many copies the profile's Test starts; 0 forgets the choice.
func (s *Store) SetTestCopies(game, id string, n int) (Profile, error) {
	if n < 0 || n > MaxTestCopies {
		return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("start 1 to %d copies", MaxTestCopies))
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.TestCopies = n
		return nil
	})
}

// SetModOrder remembers the order the Mods tab shows entries in under its Manual sort. Keys of entries the profile
// does not have are dropped; a repeated key is an error.
func (s *Store) SetModOrder(game, id string, keys []string) (Profile, error) {
	seen := make(map[string]bool, len(keys))
	for _, k := range keys {
		if seen[k] {
			return Profile{}, usererr.Wrap(usererr.Invalid, fmt.Errorf("mod order lists %q twice", k))
		}
		seen[k] = true
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		have := make(map[string]bool, len(p.Entries))
		for _, e := range p.Entries {
			have[e.Key] = true
		}
		p.ModOrder = slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return !have[k] })
		return nil
	})
}

// SetLaunchOptions replaces a profile's extra SMAPI arguments. It never touches mods/, so a running
// game does not block it.
func (s *Store) SetLaunchOptions(game, id, options string) (Profile, error) {
	if _, err := gamereg.ParseLaunchOptions(game, s.LoaderID(game, id), options); err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.LaunchOptions = options
		return nil
	})
}
