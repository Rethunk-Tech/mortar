// Package mod identifies mods across games: an ID names a mod by the format that defines it and the id that format uses.
package mod

import (
	"strings"
	"unicode"
)

// Mod formats.
const (
	FormatSMAPI        = "smapi"
	FormatBepInEx      = "bepinex"
	FormatMelon        = "melon"
	FormatThunderstore = "thunderstore"
	// FormatFolder names a mod of a game with no loader, by the store item that holds its files.
	FormatFolder = "folder"
	// FormatModInfo is a folder game's mod by the id its mod-root file gives (Vintage Story's modinfo.json modid),
	// as a save names it.
	FormatModInfo = "modinfo"
)

// ID is "<format>:<local>", for example "smapi:Pathoschild.ContentPatcher". The zero ID is empty.
type ID string

// NewID joins a format and a format-local id.
func NewID(format, local string) ID { return ID(format + ":" + local) }

// SMAPI is shorthand for NewID(FormatSMAPI, uniqueID).
func SMAPI(uniqueID string) ID { return NewID(FormatSMAPI, uniqueID) }

// Parse reads user-typed text: a full id as is, a bare local id in defaultFormat.
func Parse(s, defaultFormat string) ID {
	s = strings.TrimSpace(s)
	if strings.Contains(s, ":") {
		return ID(s)
	}
	return NewID(defaultFormat, s)
}

// Format is the part before the first colon; empty when the ID has none.
func (id ID) Format() string {
	f, _, ok := strings.Cut(string(id), ":")
	if !ok {
		return ""
	}
	return f
}

// Local is the part after the first colon, or the whole text when there is no colon.
func (id ID) Local() string {
	_, l, ok := strings.Cut(string(id), ":")
	if !ok {
		return string(id)
	}
	return l
}

// Fold is the key an ID is mapped and compared by: SMAPI ids are case-insensitive and may carry stray spaces.
func (id ID) Fold() string {
	if id.Format() == FormatSMAPI {
		return FormatSMAPI + ":" + strings.ToLower(strings.TrimSpace(id.Local()))
	}
	return string(id)
}

// Equal reports whether two ids name the same mod: formats compare exactly, locals by the format's rule.
func Equal(a, b ID) bool {
	if a.Format() == FormatSMAPI && b.Format() == FormatSMAPI {
		return strings.EqualFold(strings.TrimSpace(a.Local()), strings.TrimSpace(b.Local()))
	}
	return a == b
}

// Strings returns the ids as plain strings.
func Strings(ids []ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

// Locals returns the local part of each id.
func Locals(ids []ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.Local()
	}
	return out
}

// LooseName keeps only s's letters and digits, lower-cased, so two names that differ in case, spacing or punctuation
// compare equal.
func LooseName(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}
