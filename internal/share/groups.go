package share

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// FileGroup is a profile group in a .mortar file, named by entry identity rather than store keys.
type FileGroup struct {
	Name string `json:"name"`
	Refs []Ref  `json:"refs,omitempty"`
}

func collectFileGroups(p profile.Profile) []FileGroup {
	out := make([]FileGroup, 0, len(p.Groups))
	for _, g := range p.Groups {
		fg := FileGroup{Name: g.Name}
		for _, key := range g.Keys {
			i := slices.IndexFunc(p.Entries, func(e profile.Entry) bool { return e.Key == key })
			if i < 0 {
				continue
			}
			id, ok := identityOf(p.Entries[i])
			if !ok {
				continue
			}
			if !slices.ContainsFunc(fg.Refs, func(r Ref) bool { return r.MatchesEntry(p.Entries[i]) }) {
				fg.Refs = append(fg.Refs, id)
			}
		}
		out = append(out, fg)
	}
	return out
}

func identityOf(e profile.Entry) (Ref, bool) {
	switch e.Source.Kind {
	case profile.KindNexus:
		if e.Source.ModID <= 0 || e.Source.FileID <= 0 {
			return Ref{}, false
		}
		return Ref{ModID: e.Source.ModID, FileID: e.Source.FileID}, true
	case profile.KindGitHub:
		if e.Source.Repo == "" || e.Source.Tag == "" || e.Source.Asset == "" {
			return Ref{}, false
		}
		return Ref{GitHub: e.Source.Repo + "@" + e.Source.Tag + "/" + e.Source.Asset}, true
	case profile.KindPatreon:
		if e.Source.Name == "" {
			return Ref{}, false
		}
		return Ref{Patreon: e.Source.Name}, true
	case profile.KindThunderstore:
		if e.Source.Name == "" {
			return Ref{}, false
		}
		return Ref{Package: e.Source.Name, Version: e.Source.Version}, true
	case profile.KindLocal:
		return Ref{Local: e.StoreKey(), LocalName: e.Source.Name}, true
	default:
		return Ref{}, false
	}
}

// ResolveGroupKeys maps a file group's refs onto the profile's current entry keys.
func ResolveGroupKeys(p profile.Profile, g FileGroup) []string {
	var keys []string
	for _, id := range g.Refs {
		for _, e := range p.Entries {
			if id.MatchesEntry(e) && !slices.Contains(keys, e.Key) {
				keys = append(keys, e.Key)
			}
		}
	}
	return keys
}
