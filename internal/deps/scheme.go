package deps

import (
	"github.com/Rethunk-Tech/go-semverish"
	"github.com/Rethunk-Tech/go-smapi"
)

// Version scheme names a loader reports as its capability.
const (
	SemverSMAPI  = "semver-smapi"
	SemverStrict = "semver-strict"
	Opaque       = "opaque"
)

// Status is whether a version meets a constraint: Met, Unmet, or Unknown when the two cannot be ordered.
type Status = smapi.Status

const (
	Unknown = smapi.Unknown
	Met     = smapi.Met
	Unmet   = smapi.Unmet
)

// Compare orders two versions under the named scheme: -1, 0 or 1. ok is false when the scheme (opaque, or unknown) or
// the versions cannot be ordered. Both semver schemes order leniently (a leading v, two to four numbers), since a
// version mods declare in a strict scheme is still often written loosely.
func Compare(scheme, a, b string) (cmp int, ok bool) {
	if scheme != SemverSMAPI && scheme != SemverStrict {
		return 0, false
	}
	return semverish.Compare(a, b)
}

// Check says whether version meets the minimum constraint under the named scheme; an unknown scheme is opaque. An
// empty constraint is Met. A pair a semver scheme cannot order is Unknown, as is any opaque version other than the
// constraint itself, since an opaque scheme cannot tell newer from older.
func Check(scheme, version, constraint string) Status {
	if scheme == SemverSMAPI || scheme == SemverStrict {
		return smapi.Dependency{MinimumVersion: constraint}.Check(version)
	}
	if constraint == "" || version == constraint {
		return Met
	}
	return Unknown
}
