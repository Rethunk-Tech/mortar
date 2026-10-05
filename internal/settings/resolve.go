package settings

import (
	"maps"
	"strings"
)

// ResolveAt returns the value of key at sc, most specific first: the profile's override (profile holds the
// overrides of sc.Profile), then the key's own scope (the game's block, the source's, the loader's or the app's), then the
// registry default. Install is walked once a key lives there.
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
	case ScopeSource, ScopeLoader:
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

// LoaderPrefs is what Mortar keeps for one loader, whichever game uses it.
type LoaderPrefs struct {
	// Pin is the loader version to stay on; empty follows the latest release.
	Pin string `json:"pin,omitempty"`
	// LastSweepVersion is the loader version the last patch-day sweep ran with.
	LastSweepVersion string `json:"lastSweepVersion,omitempty"`
}

// LoaderPin is the version the loader is pinned to, "" when it follows the latest release.
func (s Settings) LoaderPin(loaderID string) string { return s.LoaderPrefs[loaderID].Pin }

// PinKey is the setting that pins the loader's version, false for a loader that has none.
func PinKey(loaderID string) (string, bool) {
	for _, p := range registry {
		if p.spec.Loader == loaderID && strings.HasSuffix(p.spec.Key, "Pin") {
			return p.spec.Key, true
		}
	}
	return "", false
}

func setLoaderPrefs(s *Settings, loaderID string, edit func(*LoaderPrefs)) {
	next := maps.Clone(s.LoaderPrefs)
	if next == nil {
		next = map[string]LoaderPrefs{}
	}
	p := next[loaderID]
	edit(&p)
	if p == (LoaderPrefs{}) {
		delete(next, loaderID)
	} else {
		next[loaderID] = p
	}
	s.LoaderPrefs = next
}

func pinPref(key, loaderID string) pref {
	return strPref(key, ScopeLoader, func(s Settings, _ string) string { return s.LoaderPin(loaderID) }, func(s *Settings, _, v string) {
		setLoaderPrefs(s, loaderID, func(p *LoaderPrefs) { p.Pin = v })
	})
}

// loaderPref binds a key to the mod loader it belongs to.
func loaderPref(loader string, p pref) pref {
	p.spec.Loader = loader
	return p
}
