package meta

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/Rethunk-Tech/go-nexusmods"
	"github.com/Rethunk-Tech/go-safefs/diskcache"
)

const collectionTTL = 15 * time.Minute

// nexusClient is the credential-less Nexus client collections are read through; it keeps one rate-limit budget.
func (c *Client) nexusClient() *nexusmods.Client {
	c.nexusOnce.Do(func() {
		c.nexus = nexusmods.New("Mortar", c.Version)
		c.nexus.HTTP, c.nexus.BaseURL = c.client(), c.NexusURL
	})
	return c.nexus
}

// Collection reads a collection revision, cached for 15 minutes.
func (c *Client) Collection(ctx context.Context, domain, slug string, revision int) (nexusmods.Collection, error) {
	col, err := diskcache.Cached(c.Cache(), CollectionPrefix+domain+"-"+slug+"-"+strconv.Itoa(revision)+".json", collectionTTL, func() (nexusmods.Collection, error) {
		return c.nexusClient().Collection(ctx, domain, slug, revision)
	})
	// A collection cached before links were limited to web pages still holds whatever scheme the curator wrote.
	for i := range col.External {
		col.External[i].URL = WebLink(col.External[i].URL)
	}
	return col, err
}

// WebLink is raw when it is an http or https address with a host, else empty: a remote link field is shown to the
// player as a link and never followed by Mortar, and the import dialog opens it with the system's URL handler, which
// would launch other schemes' apps.
func WebLink(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return ""
	}
	return raw
}
