package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const damagedAsidePrefix = fileName + ".damaged-"

var errNoSnapshot = errors.New("no history snapshot to rebuild from")

// Repair rebuilds profile.json from the latest readable history snapshot and keeps mods/.
func (s *Store) Repair(game, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if _, err := readAt(dir, id); err == nil {
		return Profile{}, fmt.Errorf("profile %s is not damaged", id)
	}
	entries, ok := latestSnapshotAt(dir)
	if !ok {
		return Profile{}, errNoSnapshot
	}
	src := filepath.Join(dir, fileName)
	broken, _ := fsx.ReadFile(src)
	aside := filepath.Join(dir, fmt.Sprintf("%s%d", damagedAsidePrefix, time.Now().UnixNano()))
	if err := fsx.Rename(src, aside); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Profile{}, err
	}
	p := salvageProfile(broken, id)
	p.Entries = entries
	p.Updated = time.Now().UTC().Truncate(time.Second)
	if err := datadir.WriteJSON(src, p); err != nil {
		_ = fsx.Rename(aside, src)
		return Profile{}, err
	}
	return p, nil
}

// UndoRepair puts the aside damaged profile.json back.
func (s *Store) UndoRepair(game, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	aside, err := latestDamagedAside(dir)
	if err != nil {
		return err
	}
	src := filepath.Join(dir, fileName)
	_ = os.Remove(src)
	return fsx.Rename(aside, src)
}

func salvageProfile(b []byte, id string) Profile {
	var p Profile
	_ = json.NewDecoder(bytes.NewReader(b)).Decode(&p)
	p.ID = id
	p.Error = ""
	p.RepairError = ""
	p.Entries = nil
	if p.Name == "" {
		p.Name = id
	}
	if p.Created.IsZero() {
		p.Created = time.Now().UTC().Truncate(time.Second)
	}
	return p
}

func latestDamagedAside(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, damagedAsidePrefix+"*"))
	if err != nil {
		return "", err
	}
	var best string
	var bestMod time.Time
	for _, path := range matches {
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		if best == "" || fi.ModTime().After(bestMod) {
			best, bestMod = path, fi.ModTime()
		}
	}
	if best == "" {
		return "", fmt.Errorf("no damaged profile.json aside to restore")
	}
	return best, nil
}
