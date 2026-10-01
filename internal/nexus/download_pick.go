package nexus

import "slices"

// PickDownloadLink returns the mirror to fetch: preferredShortName when it matches a link's ShortName, else the first entry.
func PickDownloadLink(links []Link, preferredShortName string) (Link, bool) {
	if len(links) == 0 {
		return Link{}, false
	}
	if preferredShortName != "" {
		if i := slices.IndexFunc(links, func(l Link) bool { return l.ShortName == preferredShortName }); i >= 0 {
			return links[i], true
		}
	}
	return links[0], true
}
