package nexus

import (
	"slices"

	"github.com/Rethunk-Tech/go-nexusmods"
)

// PickDownloadLink returns the mirror to fetch: preferredShortName when it matches a link's ShortName, else the first entry.
func PickDownloadLink(links []nexusmods.Link, preferredShortName string) (nexusmods.Link, bool) {
	if len(links) == 0 {
		return nexusmods.Link{}, false
	}
	if preferredShortName != "" {
		if i := slices.IndexFunc(links, func(l nexusmods.Link) bool { return l.ShortName == preferredShortName }); i >= 0 {
			return links[i], true
		}
	}
	return links[0], true
}
