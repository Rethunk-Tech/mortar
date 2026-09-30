// Package settings persists Mortar's user preferences as settings.json.
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/datadir"
)

const fileName = "settings.json"

var accents = []string{"sand", "moss", "copper", "sky"}

// Settings is the on-disk shape of settings.json.
type Settings struct {
	Accent      string `json:"accent"`
	Translucent bool   `json:"translucent"`
	LastGame    string `json:"lastGame"`
}

// Defaults returns the settings used when no valid file exists.
func Defaults() Settings {
	return Settings{Accent: "sand", Translucent: true}
}

// Store reads and writes settings.json under the user data folder.
type Store struct {
	mu   sync.Mutex
	path string
	cur  Settings
}

// Open loads settings from the data folder; a missing or corrupt file yields defaults.
func Open() (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, fileName), cur: Defaults()}
	if b, err := os.ReadFile(s.path); err == nil {
		var loaded Settings
		if json.Unmarshal(b, &loaded) == nil {
			s.cur = loaded
		}
	}
	if !slices.Contains(accents, s.cur.Accent) {
		s.cur.Accent = Defaults().Accent
	}
	return s, nil
}

// Get returns the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Update applies fn to a copy of the settings, validates, persists atomically and returns the result.
func (s *Store) Update(fn func(*Settings)) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.cur
	fn(&next)
	if !slices.Contains(accents, next.Accent) {
		return s.cur, fmt.Errorf("unknown accent %q", next.Accent)
	}
	if err := writeAtomic(s.path, next); err != nil {
		return s.cur, err
	}
	s.cur = next
	return next, nil
}

func writeAtomic(path string, v Settings) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), fileName+".*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(f.Name()))
		}
	}()
	if _, err = f.Write(b); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
