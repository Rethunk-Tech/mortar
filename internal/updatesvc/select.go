package updatesvc

import (
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// preferRelease picks which updater result to offer. When includeBeta is false, prereleases are ignored.
func preferRelease(includeBeta bool, stable, beta *updater.Release) *updater.Release {
	if !includeBeta {
		return stable
	}
	if stable == nil {
		return beta
	}
	if beta == nil {
		return stable
	}
	if versionNewer(beta.Version, stable.Version) {
		return beta
	}
	return stable
}

func versionNewer(a, b string) bool {
	cmp, ok := meta.CompareVersions(a, b)
	return ok && cmp > 0
}
