package problems

import (
	"context"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/source"
	"github.com/Rethunk-Tech/mortar/internal/source/thunderstore"
)

// DeprecatedPackage is an installed Thunderstore package its author deprecated. Replacement is the package the
// deprecated one's own summary points to ("Namespace-Name"), empty when it names none.
type DeprecatedPackage struct {
	Key         string `json:"key"`
	ID          mod.ID `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Replacement string `json:"replacement,omitempty"`
}

// deprecationSource lists a community's deprecated packages; the Thunderstore driver implements it.
type deprecationSource interface {
	Deprecated(ctx context.Context, key, version string) (map[string]thunderstore.Deprecation, error)
}

// deprecatedPackages matches the installed packages against the community's deprecated ones. A failed index read
// finds none: this is advice, never an error.
func deprecatedPackages(ctx context.Context, src deprecationSource, key, version string, pkgs []profile.PackageRef) []DeprecatedPackage {
	if key == "" || len(pkgs) == 0 {
		return nil
	}
	dep, err := src.Deprecated(ctx, key, version)
	if err != nil {
		return nil
	}
	var out []DeprecatedPackage
	for _, p := range pkgs {
		if d, ok := dep[strings.ToLower(p.Name)]; ok {
			out = append(out, DeprecatedPackage{Key: p.Key, ID: p.ID, Name: p.Name, Version: p.Version, Replacement: d.Replacement})
		}
	}
	return out
}

// thunderstoreKey is the game's Thunderstore community, empty when the catalog lists none.
func thunderstoreKey(gameID string) string {
	for _, g := range game.Catalog() {
		if g.ID == gameID {
			gs, _ := g.Source("thunderstore")
			return gs.Key
		}
	}
	return ""
}

func thunderstoreSource() (thunderstore.Driver, bool) {
	e, ok := source.Get("thunderstore")
	if !ok {
		return thunderstore.Driver{}, false
	}
	d, ok := e.Source.(thunderstore.Driver)
	return d, ok
}
