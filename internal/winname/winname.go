// Package winname applies Windows' file name rules, the strictest of the platforms Mortar runs on, so a name that is
// fine on Linux cannot later fail or silently alias another file on Windows.
package winname

import (
	"slices"
	"strings"
)

const invalid = `<>:"/\|?*`

// Valid reports whether seg, one path segment, is a name Windows stores as written: no reserved characters, no
// trailing dot or space (Win32 strips them, so "foo." would overwrite "foo"), and not a device name.
func Valid(seg string) bool {
	if seg == "" || strings.ContainsFunc(seg, func(r rune) bool { return r < ' ' || strings.ContainsRune(invalid, r) }) {
		return false
	}
	return !strings.HasSuffix(seg, ".") && !strings.HasSuffix(seg, " ") && !device(seg)
}

// Clean turns name into one valid segment: reserved characters become '-', edge dots and spaces go, and a device
// name gets an underscore after its stem ("nul.txt" becomes "nul_.txt"). An empty result is returned as "".
func Clean(name string) string {
	name = strings.Trim(strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(invalid, r) {
			return '-'
		}
		return r
	}, name), " .")
	if device(name) {
		stem, ext, dot := strings.Cut(name, ".")
		name = strings.TrimRight(stem, " ") + "_"
		if dot {
			name += "." + ext
		}
	}
	return name
}

// device reports Windows device names, which Windows opens as devices even with an extension ("CON.txt") or
// trailing spaces. COM0, LPT0 and the superscript ports (COM¹ to LPT³) are devices too.
func device(seg string) bool {
	base, _, _ := strings.Cut(seg, ".")
	base = strings.ToUpper(strings.TrimRight(base, " "))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	port, ok := strings.CutPrefix(base, "COM")
	if !ok {
		port, ok = strings.CutPrefix(base, "LPT")
	}
	return ok && slices.Contains([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "¹", "²", "³"}, port)
}
