package share

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/profile"
)

type groupRef struct {
	ModID  int    `json:"modId,omitempty"`
	FileID int    `json:"fileId,omitempty"`
	GitHub string `json:"github,omitempty"`
}

// FileGroup is a profile group in a .mortar file, named by entry identity rather than store keys.
type FileGroup struct {
	Name string     `json:"name"`
	Refs []groupRef `json:"refs,omitempty"`
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
			fg.Refs = append(fg.Refs, id)
		}
		out = append(out, fg)
	}
	return out
}

func identityOf(e profile.Entry) (groupRef, bool) {
	switch e.Source.Kind {
	case profile.KindNexus:
		if e.Source.ModID <= 0 || e.Source.FileID <= 0 {
			return groupRef{}, false
		}
		return groupRef{ModID: e.Source.ModID, FileID: e.Source.FileID}, true
	case profile.KindGitHub:
		if e.Source.Repo == "" || e.Source.Tag == "" || e.Source.Asset == "" {
			return groupRef{}, false
		}
		return groupRef{GitHub: e.Source.Repo + "@" + e.Source.Tag + "/" + e.Source.Asset}, true
	default:
		return groupRef{}, false
	}
}

// ResolveGroupKeys maps a file group's refs onto the profile's current entry keys.
func ResolveGroupKeys(p profile.Profile, g FileGroup) []string {
	var keys []string
	for _, id := range g.Refs {
		r := Ref{ModID: id.ModID, FileID: id.FileID, GitHub: id.GitHub}
		for _, e := range p.Entries {
			if r.MatchesEntry(e) && !slices.Contains(keys, e.Key) {
				keys = append(keys, e.Key)
			}
		}
	}
	return keys
}
