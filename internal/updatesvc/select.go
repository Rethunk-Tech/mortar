package updatesvc

import (
	"github.com/Rethunk-Tech/mortar/internal/meta"
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
	if meta.Newer(beta.Version, stable.Version) {
		return beta
	}
	return stable
}
