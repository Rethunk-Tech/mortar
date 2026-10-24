package source

import (
	"context"
	"net/url"
	"time"
)

// Details is the fuller page of one mod, read when the player opens it in Browse.
type Details struct {
	// Description is the mod's long text as Markdown (its README); empty when the site has none.
	Description string `json:"description"`
	// Changelog is the site's changelog as Markdown, when it keeps one apart from its versions.
	Changelog string `json:"changelog,omitempty"`
	// Versions are the published version numbers, newest first.
	Versions []string `json:"versions"`
	// Released holds when each of Versions was published, as Unix seconds at the same index, 0 where the site gave no
	// readable date; empty when the site dates none.
	Released []int64 `json:"released,omitempty"`
	// Dependencies are the ids the newest version needs, as the site names them.
	Dependencies []string `json:"dependencies"`
	Categories   []string `json:"categories"`
	// Donation is the author's donation page, an http or https address; empty when the site lists none.
	Donation string `json:"donation,omitempty"`
}

// WebLink is raw when it is an http or https address with a host, else empty: a site's own link field is remote data,
// shown to the player as a link and never followed by Mortar.
func WebLink(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return ""
	}
	return raw
}

// Detailer is a source that reads one mod's Details; key is the source's name for the game.
type Detailer interface {
	Details(ctx context.Context, key, id, mortarVersion string) (Details, error)
}

// Released is an RFC 3339 time as Unix seconds, or 0 when it does not parse.
func Released(rfc3339 string) int64 {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return 0
	}
	return t.Unix()
}
