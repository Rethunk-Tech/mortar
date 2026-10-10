package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
)

// ModSetHash identifies the files the profile's enabled packages lay out below the profile's root: each file's path
// with the key of the entry that wins it. An entry's key names an immutable store item (and file), so the hash needs no
// file contents and changes exactly when the enabled set, an update or a rollback changes what is laid out.
func (s *Store) ModSetHash(game, id string) (string, error) {
	files, _, err := s.packageFiles(game, id)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		h.Write([]byte(rel + "\x00" + files[rel].key + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// CacheSet is the mod set hash the profile's caches were last cleared for, or "" when they never were.
func (s *Store) CacheSet(game, id string) string {
	p, err := s.read(game, id)
	if err != nil {
		return ""
	}
	return p.CacheSet
}

// SetCacheSet records the mod set hash the profile's caches were just cleared for. A running game does not block it.
func (s *Store) SetCacheSet(game, id, hash string) error {
	_, err := s.update(game, id, func(p *Profile, _ string) error {
		p.CacheSet = hash
		return nil
	})
	return err
}
