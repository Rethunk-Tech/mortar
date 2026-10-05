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
	Required     Relation = "required"
	Optional     Relation = "optional"
	Incompatible Relation = "incompatible"
	// Embedded is a dependency the owner ships inside itself.
	Embedded Relation = "embedded"
	// Tool is something the user runs beside the game, not a component.
	Tool Relation = "tool"
	// Include pulls another package's files in with the owner's.
	Include Relation = "include"
)

// Target is what a dependency points at: a component when the owner's format names one, else a source's package.
type Target struct {
	Mod mod.ID
	// Package is "<source>:<native id>" for a dependency the owner names only by the source's package.
	Package string
}

func (t Target) String() string {
	if t.Mod != "" {
		return string(t.Mod)
	}
	return t.Package
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
	parts := strings.Split(s, "-")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
		return Dependency{}, fmt.Errorf("malformed dependency %q", s)
	}
	return Dependency{
		Target:     Target{Package: "thunderstore:" + parts[0] + "-" + parts[1]},
		Constraint: parts[2],
		Relation:   Required,
	}, nil
}

// SMAPI maps a manifest Dependencies[] entry; required follows IsRequired.
func SMAPI(uniqueID, minimumVersion string, required bool) Dependency {
	rel := Required
	if !required {
		rel = Optional
	}
	return Dependency{Target: Target{Mod: mod.SMAPI(uniqueID)}, Constraint: minimumVersion, Relation: rel}
}
