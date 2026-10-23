package source

import "context"

// Facts is what a source's cache holds of one package and of the version of it that is installed. A figure the cache
// lacks stays empty or zero.
type Facts struct {
	// Released is when the installed version was published (RFC 3339), empty when the cache lacks that version or its date.
	Released string
	// Latest is the newest version the update check offers, and LatestReleased its date.
	Latest, LatestReleased string
	// Updated is the package's own last-update date as the site gives it.
	Updated string
	Rating  int
	// Downloads is the package's total over all its versions.
	Downloads  int
	Categories []string
}

// CachedFacter is a source that answers Facts from what it already cached, never asking the network. installed maps
// each package ref to its installed version, and the answer holds the refs the cache knows, under the same spelling.
// key is the source's name for the game, empty for a source that has none.
type CachedFacter interface {
	CachedFacts(ctx context.Context, key string, installed map[string]string) (map[string]Facts, error)
}
