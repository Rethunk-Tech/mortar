package problems

import (
	"context"

	cp "github.com/Rethunk-Tech/go-contentpatcher"
	"github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher"
)

type cachedIndex struct {
	fingerprint string
	targets     []cp.AssetTarget
}

//wails:ignore
func (s *Service) WhoChanges(_ context.Context, gameID, id, query string) (cp.WhoChangesPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return cp.WhoChangesPage{}, err
	}
	return cp.WhoChangesOf(index, query), nil
}

// AssetMap pages the profile's touched assets; shared keeps only those more than one mod changes.
func (s *Service) AssetMap(_ context.Context, gameID, id, filter string, shared bool, offset int) (cp.AssetMapPage, error) {
	index, err := s.assetIndex(gameID, id)
	if err != nil {
		return cp.AssetMapPage{}, err
	}
	return cp.AssetMapOf(index, filter, shared, offset), nil
}

// assetIndex builds the profile's index on first use and keeps it until the problems fingerprint changes, so
// searching and paging do not re-read every content pack.
func (s *Service) assetIndex(gameID, id string) ([]cp.AssetTarget, error) {
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
