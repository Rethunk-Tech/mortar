// Package deps models what one component needs from another, in the owner's own version scheme, and maps each
// source's dependency vocabulary into that model.
package deps

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Relation is how a dependency binds its owner.
type Relation string

const (
	Required Relation = "required"
	Optional Relation = "optional"
)

// Target is what a dependency points at: a component when the owner's format names one, else a source's package.
type Target struct {
	Mod mod.ID
	// Package is "<source>:<native id>" for a dependency the owner names only by the source's package.
	Package string
}

// Dependency is one relation from an owner to a target. Constraint is read by the owner's version scheme: the
// minimum version for the semver schemes, the exact version for opaque; empty accepts any.
type Dependency struct {
	Target     Target
	Constraint string
	Relation   Relation
}

// Thunderstore reads a "Namespace-Name-Version" dependency string; the target keeps the spelling.
func Thunderstore(s string) (Dependency, error) {
	pkg, version, ok := ThunderstoreRef(s)
	if !ok {
		return Dependency{}, fmt.Errorf("malformed dependency %q", s)
	}
	return Dependency{Target: Target{Package: "thunderstore:" + pkg}, Constraint: version, Relation: Required}, nil
}

// ThunderstoreRef splits "Namespace-Name-Version" into its "Namespace-Name" package and its version. Thunderstore
// allows no '-' inside a namespace, name or version, so anything but three parts with a namespace and name is malformed.
func ThunderstoreRef(s string) (pkg, version string, ok bool) {
	parts := strings.Split(s, "-")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0] + "-" + parts[1], parts[2], true
}

// SMAPI maps a manifest Dependencies[] entry; required follows IsRequired.
func SMAPI(uniqueID, minimumVersion string, required bool) Dependency {
	rel := Required
	if !required {
		rel = Optional
	}
	return Dependency{Target: Target{Mod: mod.SMAPI(uniqueID)}, Constraint: minimumVersion, Relation: rel}
}
