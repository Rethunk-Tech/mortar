// Package mod identifies mods across games: an ID names a mod by the format that defines it and the id that format uses.
package mod

import "strings"

// Mod formats. SMAPI is the only one with a reader today.
const (
	FormatSMAPI        = "smapi"
	FormatBepInEx      = "bepinex"
	FormatThunderstore = "thunderstore"
)

// ID is "<format>:<local>", for example "smapi:Pathoschild.ContentPatcher". The zero ID is empty.
type ID string

// NewID joins a format and a format-local id.
func NewID(format, local string) ID { return ID(format + ":" + local) }

// SMAPI is shorthand for NewID(FormatSMAPI, uniqueID).
func SMAPI(uniqueID string) ID { return NewID(FormatSMAPI, uniqueID) }

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
func Equal(a, b ID) bool { return a.Fold() == b.Fold() }

// Strings returns the ids as plain strings.
func Strings(ids []ID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}
