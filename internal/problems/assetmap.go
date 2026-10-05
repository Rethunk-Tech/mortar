package problems

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
)

type cachedIndex struct {
	fingerprint string
	targets     []contentpatcher.AssetTarget
}

//wails:ignore
func (s *Service) WhoChanges(_ context.Context, gameID, id, query string) (contentpatcher.WhoChangesPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return contentpatcher.WhoChangesPage{}, err
	}
	return contentpatcher.WhoChangesOf(index, query), nil
}

// AssetMap pages the profile's touched assets; shared keeps only those more than one mod changes.
func (s *Service) AssetMap(_ context.Context, gameID, id, filter string, shared bool, offset int) (contentpatcher.AssetMapPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return contentpatcher.AssetMapPage{}, err
	}
	return contentpatcher.AssetMapOf(index, filter, shared, offset), nil
}

// assetIndex builds the profile's index on first use and keeps it until the problems fingerprint changes, so
// searching and paging do not re-read every content pack.
func (s *Service) assetIndex(gameID, id string) ([]contentpatcher.AssetTarget, error) {
	mods, err := s.installed(gameID, id)
	if err != nil {
		return nil, err
	}
	key := gameID + "/" + id
	fp := fingerprint(Environment{}, mods, "")
	s.mu.Lock()
	c, ok := s.assets[key]
	s.mu.Unlock()
	if ok && c.fingerprint == fp {
		return c.targets, nil
	}
	index := contentpatcher.BuildAssetIndex(mods)
	s.mu.Lock()
	if s.assets == nil {
		s.assets = map[string]cachedIndex{}
	}
	s.assets[key] = cachedIndex{fingerprint: fp, targets: index}
	s.mu.Unlock()
	return index, nil
}
