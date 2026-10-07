package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// DependencyNames maps each id of ids to the name of the Nexus page that ships it, for a dependency that is not
// installed and so has no name of its own. An id the dataset does not know, or whose page cannot be read, is left out.
func DependencyNames(ctx context.Context, m Meta, ids []string) map[string]string {
	out := map[string]string{}
	for _, id := range ids {
		refs, err := m.Lookup(ctx, mod.ID(id).Local())
		if err != nil {
			continue
		}
		for _, r := range refs {
			if !strings.EqualFold(r.Site, "nexus") {
				continue
			}
			if page, err := m.Page(ctx, r.ID); err == nil && page.Name != "" {
				out[id] = page.Name
				break
			}
		}
	}
	return out
}

// DependencyNames names uninstalled dependencies by their Nexus page.
func (s *Service) DependencyNames(ctx context.Context, gameID string, ids []string) map[string]string {
	if s.meta == nil {
		return map[string]string{}
	}
	return DependencyNames(ctx, s.metaFor(gameID), ids)
}
