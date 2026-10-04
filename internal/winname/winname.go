// Package winname applies Windows' file name rules, the strictest of the platforms Mortar runs on, so a name that is
// fine on Linux cannot later fail or silently alias another file on Windows.
package winname

import "strings"

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
// name gets a trailing underscore. An empty result is returned as "".
func Clean(name string) string {
	name = strings.Trim(strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(invalid, r) {
			return '-'
		}
		return r
	}, name), " .")
	if device(name) {
		name += "_"
	}
	return name
}

// device reports Windows device names, which Windows opens as devices even with an extension ("CON.txt") or
// trailing spaces.
func device(seg string) bool {
	base, _, _ := strings.Cut(seg, ".")
	base = strings.ToUpper(strings.TrimRight(base, " "))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9'
}
