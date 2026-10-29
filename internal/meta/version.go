package meta

import (
	"cmp"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*))?(?:\+[0-9A-Za-z.-]+)?$`)

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
		v.pre = strings.FieldsFunc(m[5], func(r rune) bool { return r == '.' || r == '-' })
	}
	return v, true
}

// Newer reports whether a is a newer version than b; either failing to parse means not newer.
func Newer(a, b string) bool {
	c, ok := CompareVersions(a, b)
	return ok && c > 0
}

// NewerFile reports whether a download file (version a, file id aID) is newer than another of the same mod: by
// version where both parse and differ, else the later upload, which has the higher file id (an author re-uploading a
// fix under the same version).
func NewerFile(a string, aID int64, b string, bID int64) bool {
	if c, ok := CompareVersions(a, b); ok && c != 0 {
		return c > 0
	}
	return aID > bID
}

// CompareVersions orders two SMAPI semantic versions (major.minor, optional patch and fourth number,
// optional -prerelease and +build): -1, 0 or 1. ok is false when either is not one. A release outranks its
// prereleases, build metadata is ignored, '.' and '-' both separate prerelease identifiers, and an "unofficial"
// identifier ranks below any other.
func CompareVersions(a, b string) (c int, ok bool) {
	x, okA := parseVersion(a)
	y, okB := parseVersion(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range x.nums {
		if x.nums[i] != y.nums[i] {
			return cmp.Compare(x.nums[i], y.nums[i]), true
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
	return cmp.Compare(len(x.pre), len(y.pre)), true
}

// comparePre orders two prerelease identifiers: "unofficial" lowest, then numeric ones by value (any length), which
// rank below alphanumeric ones, then text ignoring case.
func comparePre(a, b string) int {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if a == b {
		return 0
	}
	switch {
	case b == "unofficial":
		return 1
	case a == "unofficial":
		return -1
	}
	na, nb := isDigits(a), isDigits(b)
	switch {
	case na && nb:
		a, b = strings.TrimLeft(a, "0"), strings.TrimLeft(b, "0")
		return cmp.Or(cmp.Compare(len(a), len(b)), strings.Compare(a, b))
	case na:
		return -1
	case nb:
		return 1
	}
	return strings.Compare(a, b)
}

func isDigits(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' })
}

// CompareDotNet orders two versions the way .NET's System.Version does, which BepInEx 5 uses: two to four plain
// numbers, a part left out ranks below 0 (so 1.0 < 1.0.0), and anything else (a suffix, a leading v, one number) is
// not a version. ok is false when either is not one.
func CompareDotNet(a, b string) (c int, ok bool) {
	x, okA := parseDotNet(a)
	y, okB := parseDotNet(b)
	if !okA || !okB {
		return 0, false
	}
	return slices.Compare(x[:], y[:]), true
}

func parseDotNet(s string) (v [4]int, ok bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return v, false
	}
	for i := range v {
		v[i] = -1
		if i >= len(parts) {
			continue
		}
		n, err := strconv.ParseInt(strings.Trim(parts[i], " \t\n\v\f\r"), 10, 32)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = int(n)
	}
	return v, true
}
