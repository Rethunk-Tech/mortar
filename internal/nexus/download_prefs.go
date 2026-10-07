package nexus

import "slices"

// DownloadServers connects a client to the app's shared settings: Remember records the mirrors a file offered and
// Preferred returns the short name the user picked, empty for none. Both are optional.
type DownloadServers struct {
	Remember  func(shortNames []string)
	Preferred func() string
}

func (c *Client) applyDownloadPreferences(links []Link) []Link {
	if len(links) == 0 || c.Servers.Preferred == nil {
		return links
	}
	if c.Servers.Remember != nil {
		shortNames := make([]string, len(links))
		for i, l := range links {
			shortNames[i] = l.ShortName
		}
		c.Servers.Remember(shortNames)
	}
	picked, _ := PickDownloadLink(links, c.Servers.Preferred())
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
