// Package winname applies Windows' file name rules, the strictest of the platforms Mortar runs on, so a name that is
// fine on Linux cannot later fail or silently alias another file on Windows.
package winname

import (
	"strings"

	ga "github.com/Rethunk-Tech/go-archive"
)

const invalid = `<>:"/\|?*`

// Valid reports whether seg, one path segment, is a name Windows stores as written: no reserved characters, no
// trailing dot or space (Win32 strips them, so "foo." would overwrite "foo"), and not a device name.
func Valid(seg string) bool { return ga.WindowsSegment(seg) }

// Clean turns name into one valid segment: reserved characters become '-', edge dots and spaces go, and a device
// name gets an underscore after its stem ("nul.txt" becomes "nul_.txt"). An empty result is returned as "".
func Clean(name string) string {
	name = strings.Trim(strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(invalid, r) {
			return '-'
		}
		return r
	}, name), " .")
	// The map and trim leave a device name as the only rule a non-empty name can still fail.
	if name != "" && !Valid(name) {
		stem, ext, dot := strings.Cut(name, ".")
		name = strings.TrimRight(stem, " ") + "_"
		if dot {
			name += "." + ext
		}
	}
	return name
}
