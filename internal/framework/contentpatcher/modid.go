package contentpatcher

import "strings"

const formatSMAPI = "smapi"

// ModID is "<format>:<local>", for example "smapi:Pathoschild.ContentPatcher". The zero ModID is empty.
type ModID string

// SMAPI is shorthand for a SMAPI unique id.
func SMAPI(uniqueID string) ModID { return ModID(formatSMAPI + ":" + uniqueID) }

// Format is the part before the first colon; empty when the ID has none.
func (id ModID) Format() string {
	f, _, ok := strings.Cut(string(id), ":")
	if !ok {
		return ""
	}
	return f
}

// Local is the part after the first colon, or the whole text when there is no colon.
func (id ModID) Local() string {
	_, l, ok := strings.Cut(string(id), ":")
	if !ok {
		return string(id)
	}
	return l
}

// Fold is the key an ID is mapped and compared by: SMAPI ids are case-insensitive and may carry stray spaces.
func (id ModID) Fold() string {
	if id.Format() == formatSMAPI {
		return formatSMAPI + ":" + strings.ToLower(strings.TrimSpace(id.Local()))
	}
	return string(id)
}

// Equal reports whether two ids name the same mod: formats compare exactly, locals by the format's rule.
func Equal(a, b ModID) bool {
	if a.Format() == formatSMAPI && b.Format() == formatSMAPI {
		return strings.EqualFold(strings.TrimSpace(a.Local()), strings.TrimSpace(b.Local()))
	}
	return a == b
}
