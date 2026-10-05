package settings

// ResolveAt returns the value of key at sc, most specific first: the profile's override (profile holds the
// overrides of sc.Profile), then the key's own scope (the game's block, the source's, or the app's), then the
// registry default. Install and Loader are walked once a key lives there.
func ResolveAt(s Settings, key string, sc Scope, profile map[string]string) string {
	p, ok := lookupPref(key)
	if !ok {
		return ""
	}
	if v, set := profile[key]; set && p.spec.ProfileOverridable {
		return v
	}
	switch p.spec.Scope {
	case ScopeGame:
		if sc.Game != "" {
			return p.get(s, sc.Game)
		}
	case ScopeSource:
		return p.get(s, "")
	case ScopeInstall:
		// No key is stored per install yet; its value is the broader scopes'.
	default:
		return p.get(s, "")
	}
	return p.spec.Default
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

// sourcePref binds a key to the mod source it belongs to.
func sourcePref(source string, p pref) pref {
	p.spec.Source = source
	return p
}
