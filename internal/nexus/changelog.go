package nexus

import "github.com/Rethunk-AI/mortar/internal/meta"

// ChangelogBetween keeps entries strictly newer than installed and not newer than latest.
// all is already newest-first; that order is preserved.
func ChangelogBetween(all []Changelog, installed, latest string) []Changelog {
	out := make([]Changelog, 0, len(all))
	for _, e := range all {
		if c, ok := meta.CompareVersions(e.Version, installed); !ok || c <= 0 {
			continue
		}
		if latest != "" {
			if c, ok := meta.CompareVersions(e.Version, latest); ok && c > 0 {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}
