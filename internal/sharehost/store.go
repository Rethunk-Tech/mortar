package sharehost

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Store keeps hosted-share records under the data dir. The token is a local
// secret: it is never written into profile.json, a .mortar file, or a sync payload.
type Store struct {
	Dir string
}

var errBadKey = errors.New("sharehost: invalid game or profile id")

func validSeg(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func (s Store) file(game, profileID string) (string, error) {
	if s.Dir == "" || !validSeg(game) || !validSeg(profileID) {
		return "", errBadKey
	}
	p := filepath.Join(s.Dir, "secrets", "shares", game, profileID+".json")
	rel, err := filepath.Rel(s.Dir, p)
	if err != nil || !filepath.IsLocal(rel) {
		return "", errBadKey
	}
	return p, nil
}

// Get returns the record for a profile, or found=false when none is stored.
func (s Store) Get(game, profileID string) (Record, bool, error) {
	path, err := s.file(game, profileID)
	if err != nil {
		return Record{}, false, err
	}
	var rec Record
	found, err := datadir.ReadJSON(path, &rec)
	if err != nil || !found {
		return Record{}, found, err
	}
	return rec, true, nil
}

// Put writes the record for a profile.
func (s Store) Put(game, profileID string, rec Record) error {
	path, err := s.file(game, profileID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteJSON(path, rec)
}
