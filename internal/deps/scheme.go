package deps

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/meta"
)

// Version scheme names a loader reports as its capability.
const (
	SemverSMAPI  = "semver-smapi"
	SemverStrict = "semver-strict"
	Opaque       = "opaque"
)

// compare orders two versions: -1, 0 or 1. ok is false when the scheme cannot order them.
type compare func(a, b string) (cmp int, ok bool)

var schemes = map[string]compare{
	// SMAPI's lenient semver: two to four numbers, prerelease and build.
	SemverSMAPI: meta.CompareVersions,
	// Thunderstore's d.d.d.
	SemverStrict: compareStrict,
}

var strictPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func compareStrict(a, b string) (int, bool) {
	if !strictPattern.MatchString(a) || !strictPattern.MatchString(b) {
		return 0, false
	}
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range pa {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			if x < y {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// Compare orders two versions under the named scheme: -1, 0 or 1. ok is false when the scheme (opaque, or unknown) or
// the versions cannot be ordered.
func Compare(scheme, a, b string) (cmp int, ok bool) {
	order, known := schemes[scheme]
	if !known {
		return 0, false
	}
	return order(a, b)
}

// Satisfies reports whether version meets constraint under the named scheme; an unknown scheme is opaque. An empty
// constraint is met, and so is a pair a semver scheme cannot order, since only orderable versions are worth
// complaining about. Opaque is exact: any other version fails.
func Satisfies(scheme, version, constraint string) bool {
	if constraint == "" {
		return true
	}
	if scheme == SemverSMAPI || scheme == SemverStrict {
		c, ok := schemes[scheme](version, constraint)
		return !ok || c >= 0
	}
	return version == constraint
}
