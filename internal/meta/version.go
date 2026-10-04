package meta

import (
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)

type semver struct {
	nums [4]int
	pre  []string
}

func parseVersion(s string) (semver, bool) {
	m := versionPattern.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return semver{}, false
	}
	var v semver
	for i := range v.nums {
		if m[i+1] != "" {
			n, err := strconv.Atoi(m[i+1])
			if err != nil {
				return semver{}, false
			}
			v.nums[i] = n
		}
	}
	if m[5] != "" {
		v.pre = strings.Split(m[5], ".")
	}
	return v, true
}

// Newer reports whether a is a newer version than b; either failing to parse means not newer.
func Newer(a, b string) bool {
	c, ok := CompareVersions(a, b)
	return ok && c > 0
}

// CompareVersions orders two SMAPI semantic versions (major.minor, optional patch and fourth number,
// optional -prerelease and +build): -1, 0 or 1. ok is false when either is not one. A release outranks its
// prereleases, and build metadata is ignored.
func CompareVersions(a, b string) (cmp int, ok bool) {
	x, okA := parseVersion(a)
	y, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range x.nums {
		if x.nums[i] != y.nums[i] {
			return sign(x.nums[i] - y.nums[i]), true
		}
	}
	switch {
	case len(x.pre) == 0 && len(y.pre) == 0:
		return 0, true
	case len(x.pre) == 0:
		return 1, true
	case len(y.pre) == 0:
		return -1, true
	}
	for i := range min(len(x.pre), len(y.pre)) {
		if c := comparePre(x.pre[i], y.pre[i]); c != 0 {
			return c, true
		}
	}
	return sign(len(x.pre) - len(y.pre)), true
}

// comparePre follows semver: numeric identifiers compare as numbers and rank below alphanumeric ones.
func comparePre(a, b string) int {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	switch {
	case errA == nil && errB == nil:
		return sign(na - nb)
	case errA == nil:
		return -1
	case errB == nil:
		return 1
	}
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}
