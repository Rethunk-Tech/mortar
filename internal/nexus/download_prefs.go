package nexus

import (
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/settings"
)

func applyDownloadPreferences(links []Link) []Link {
	store, err := settings.Open()
	if err != nil || len(links) == 0 {
		return links
	}
	shortNames := make([]string, len(links))
	for i, l := range links {
		if l.ShortName != "" {
			shortNames[i] = l.ShortName
		}
	}
	_, _ = store.Update(func(v *settings.Settings) {
		settings.RememberNexusDownloadServers(v, shortNames)
	})
	cur := store.Get()
	picked, _ := PickDownloadLink(links, cur.NexusPreferredDownloadServer)
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
