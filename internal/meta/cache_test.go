package meta

import (
	"testing"
)

func TestRetiredCacheNames(t *testing.T) {
	t.Parallel()
	for rel, want := range map[string]bool{
		"github-o-r-0123456789ab-releases.json":    true,
		"github-o-r-v1-0123456789ab-notes.json":    true,
		"nexus/mod-stardewvalley-541.json":         true,
		"nexus-requirements-stardewvalley-1.json":  true,
		"github-o-r-0123456789ab-releases-v2.json": false,
		"github-o-r-v1-0123456789ab-notes-v2.json": false,
		"nexus/mod-v2-stardewvalley-541.json":      false,
		"smapi-release.json":                       false,
	} {
		if got := Retired(rel); got != want {
			t.Errorf("Retired(%q) = %v, want %v", rel, got, want)
		}
	}
}
