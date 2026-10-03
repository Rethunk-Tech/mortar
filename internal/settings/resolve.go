package settings

// Resolve returns the profile override for key when set, otherwise the game
// setting, otherwise the registry default.
func Resolve(s Settings, key, game string, profile map[string]string) string {
	if profile != nil {
		if v, ok := profile[key]; ok {
			return v
		}
	}
	if v, err := s.LookupGame(key, game); err == nil {
		return v
	}
	if p, ok := lookupPref(key); ok {
		return p.spec.Default
	}
	return ""
}

// ProfileOverridable reports whether key may be stored on a profile.
func ProfileOverridable(key string) bool {
	p, ok := lookupPref(key)
	return ok && p.spec.ProfileOverridable
}

func overridable(p pref) pref {
	p.spec.ProfileOverridable = true
	return p
}
