// Package profile stores each profile as <datadir>/profiles/<game>/<id>/profile.json beside its mods/ folder.
package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/store"
)

const (
	fileName = "profile.json"
	maxName  = 60
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// Source says where an entry came from. Kind is "local" for an archive the user picked, Name its file name.
type Source struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// EntryMod is one mod inside an entry. Folder holds its manifest.json, relative to mods/<key>/, in its
// enabled (not dot-prefixed) form; "." is the entry's own folder.
type EntryMod struct {
	UniqueID string `json:"uniqueId"`
	Version  string `json:"version"`
	Name     string `json:"name"`
	Author   string `json:"author"`
	Folder   string `json:"folder"`
}

// Entry is one mod archive in a profile. Disabled lists the UniqueIDs switched off.
type Entry struct {
	Key         string     `json:"key"`
	PreviousKey string     `json:"previousKey"`
	Source      Source     `json:"source"`
	Mods        []EntryMod `json:"mods"`
	Disabled    []string   `json:"disabled"`
}

// Profile is the on-disk shape of profile.json.
type Profile struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Notes   string    `json:"notes"`
	Cover   string    `json:"cover"`
	Order   int       `json:"order"`
	Hidden  bool      `json:"hidden"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Entries []Entry   `json:"entries"`
}

// Store reads and writes profiles under one root folder.
type Store struct {
	root  string
	trash string
	items *store.Store
	mu    sync.Mutex
	// Bundled returns the store key of the loader's bundled mods for a game, or "" when there are none.
	Bundled func(game string) string
	// Running reports whether the game is running this profile; nil means never.
	Running func(game, id string) bool
}

// RunningError is returned by operations that would change the mods/ folder of a profile its game is running.
type RunningError struct{ Game string }

func (e *RunningError) Error() string {
	name := e.Game
	if g := game.Find(e.Game); g != nil {
		name = g.Name()
	}
	return name + " is running this profile: stop the game first"
}

// unlocked returns a *RunningError while the game is running the profile.
func (s *Store) unlocked(game, id string) error {
	if s.Running != nil && s.Running(game, id) {
		return &RunningError{Game: game}
	}
	return nil
}

// AnyRunning reports whether the game is running any of its profiles.
func (s *Store) AnyRunning(game string) bool {
	all, err := s.List(game)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(all, func(p Profile) bool { return s.unlocked(game, p.ID) != nil })
}

// ModsDir returns the absolute path of the profile's mods/ folder, the one passed to the game as its mods path.
func (s *Store) ModsDir(game, id string) (string, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, "mods"))
}

// Open returns a store rooted at <datadir>/profiles, with deleted profiles in <datadir>/trash.
func Open(items *store.Store) (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	return &Store{root: filepath.Join(dir, "profiles"), trash: filepath.Join(dir, "trash"), items: items}, nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("profile name is empty")
	}
	if utf8.RuneCountInString(name) > maxName {
		return "", fmt.Errorf("profile name is longer than %d characters", maxName)
	}
	return name, nil
}

func (s *Store) gameDir(id string) (string, error) {
	if !game.Valid(id) {
		return "", fmt.Errorf("unknown game %q", id)
	}
	return filepath.Join(s.root, id), nil
}

func (s *Store) profileDir(game, id string) (string, error) {
	dir, err := s.gameDir(game)
	if err != nil {
		return "", err
	}
	if !idPattern.MatchString(id) {
		return "", fmt.Errorf("invalid profile id %q", id)
	}
	return filepath.Join(dir, id), nil
}

// List returns the game's profiles ordered by their order field.
func (s *Store) List(game string) ([]Profile, error) {
	dir, err := s.gameDir(game)
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, d := range dirs {
		if !d.IsDir() || !idPattern.MatchString(d.Name()) {
			continue
		}
		p, err := s.read(game, d.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Profile) int {
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		return a.Created.Compare(b.Created)
	})
	return out, nil
}

func (s *Store) read(game, id string) (Profile, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	return readAt(dir, id)
}

func readAt(dir, id string) (Profile, error) {
	b, err := fsx.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, fmt.Errorf("read profile %s: %w", id, err)
	}
	if p.Entries == nil {
		p.Entries = []Entry{}
	}
	for i := range p.Entries {
		if p.Entries[i].Mods == nil {
			p.Entries[i].Mods = []EntryMod{}
		}
		if p.Entries[i].Disabled == nil {
			p.Entries[i].Disabled = []string{}
		}
	}
	return p, nil
}

// Create adds a profile holding only the loader's bundled mods, if any are installed, after the existing ones.
func (s *Store) Create(game, name string) (Profile, error) {
	p, err := s.create(game, name)
	if err != nil || s.Bundled == nil {
		return p, err
	}
	key := s.Bundled(game)
	if key == "" {
		return p, nil
	}
	withBundled, err := s.AddEntry(game, p.ID, key, Source{Kind: SourceSMAPI, Name: "SMAPI"})
	if errors.Is(err, store.ErrNotFound) {
		// The item was collected while no profile used it; the next install adds it again.
		return p, nil
	}
	if err != nil {
		dir, _ := s.profileDir(game, p.ID)
		return Profile{}, errors.Join(err, os.RemoveAll(dir))
	}
	return withBundled, nil
}

func (s *Store) create(game, name string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, err := cleanName(name)
	if err != nil {
		return Profile{}, err
	}
	existing, err := s.List(game)
	if err != nil {
		return Profile{}, err
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Profile{}, err
	}
	id := hex.EncodeToString(raw[:])
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "mods"), 0o700); err != nil {
		return Profile{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	p := Profile{ID: id, Name: name, Order: len(existing), Created: now, Updated: now, Entries: []Entry{}}
	if len(existing) > 0 {
		p.Order = existing[len(existing)-1].Order + 1
	}
	if err := datadir.WriteJSON(filepath.Join(dir, fileName), p); err != nil {
		return Profile{}, errors.Join(err, os.RemoveAll(dir))
	}
	return p, nil
}

// Rename changes a profile's name and touches its updated time.
func (s *Store) Rename(game, id, name string) (Profile, error) {
	name, err := cleanName(name)
	if err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Name = name
		return nil
	})
}

// MaxNotes caps a profile's notes, in characters.
const MaxNotes = 20000

// SetNotes replaces the profile's notes. Notes never touch mods/, so a running game does not block it.
func (s *Store) SetNotes(game, id, notes string) (Profile, error) {
	if n := utf8.RuneCountInString(notes); n > MaxNotes {
		return Profile{}, fmt.Errorf("notes are too long: %d characters, the limit is %d", n, MaxNotes)
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Notes = notes
		return nil
	})
}

// update reads the profile under the lock, applies fn (given the profile folder), then writes it with a new updated time.
func (s *Store) update(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateLocked(game, id, fn)
}

func (s *Store) updateLocked(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := fn(&p, dir); err != nil {
		return Profile{}, err
	}
	p.Updated = time.Now().UTC().Truncate(time.Second)
	return p, datadir.WriteJSON(filepath.Join(dir, fileName), p)
}
