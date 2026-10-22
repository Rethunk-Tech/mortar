package problems

import (
	"context"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// nameLookups bounds the page reads in flight: a cold cache costs one network round trip per id.
const nameLookups = 16

// DependencyNames maps each id of ids to the name of the Nexus page that ships it, for a dependency that is not
// installed and so has no name of its own. An id the dataset does not know, or whose page cannot be read, is left out.
func DependencyNames(ctx context.Context, m Meta, ids []string) map[string]string {
	out := map[string]string{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, nameLookups)
	for _, id := range ids {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer func() { <-slots; wg.Done() }()
			if name := dependencyName(ctx, m, id); name != "" {
				mu.Lock()
				out[id] = name
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out
}

func dependencyName(ctx context.Context, m Meta, id string) string {
	refs, err := m.Lookup(ctx, mod.ID(id).Local())
	if err != nil {
		return ""
	}
	for _, r := range refs {
		if !strings.EqualFold(r.Site, "nexus") {
			continue
		}
		if page, err := m.Page(ctx, r.ID); err == nil && page.Name != "" {
			return page.Name
		}
	}
	return ""
}

// DependencyNames names uninstalled dependencies by their Nexus page.
func (s *Service) DependencyNames(ctx context.Context, gameID string, ids []string) map[string]string {
	if s.meta == nil {
		return map[string]string{}
	}
	return DependencyNames(ctx, s.metaFor(gameID), ids)
}
