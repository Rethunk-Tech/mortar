package nexus

import (
	"slices"

	"github.com/Rethunk-Tech/go-nexusmods"
)

// DownloadServers connects the queue to the app's shared settings: Remember records the mirrors a file offered and
// Preferred returns the short name the user picked, empty for none. Both are optional.
type DownloadServers struct {
	Remember  func(shortNames []string)
	Preferred func() string
}

// Order puts the preferred mirror first and records the ones offered, leaving links untouched.
func (s DownloadServers) Order(links []nexusmods.Link) []nexusmods.Link {
	if len(links) == 0 || s.Preferred == nil {
		return links
	}
	if s.Remember != nil {
		shortNames := make([]string, len(links))
		for i, l := range links {
			shortNames[i] = l.ShortName
		}
		s.Remember(shortNames)
	}
	picked, _ := PickDownloadLink(links, s.Preferred())
	if picked.URI == links[0].URI {
		return links
	}
	out := slices.Clone(links)
	for i, l := range out {
		if l.ShortName == picked.ShortName && l.URI == picked.URI {
			out[0], out[i] = out[i], out[0]
			break
		}
	}
	return out
}
