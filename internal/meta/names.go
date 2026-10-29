package meta

import "strings"

// Name prefixes of the cache files that expire by age. Producers build their file names from these and Clean up
// reads its expiry rules by them, so a producer's files are always claimed by a rule.
const (
	NexusDetailsPrefix    = "nexus/details-"
	NexusPagePrefix       = "nexus/page-"
	NexusCategoriesPrefix = "nexus/categories-"
	NexusChangelogsPrefix = "nexus/changelogs-"
	CollectionPrefix      = "nexus-collection-"
	StandingPrefix        = "standing-"
	DatasetPrefix         = "dataset-"
	KnownBrokenPrefix     = "known-broken-"
	LogSuppressionsPrefix = "log-suppressions-"
	// NexusModPrefix names the Nexus mod-page cache; the unversioned nexus/mod- files before it are retired.
	NexusModPrefix = "nexus/mod-v2-"
)

// RetiredCachePrefixes name files no producer writes any more; Clean up removes them whatever their age.
var RetiredCachePrefixes = []string{"nexus-requirements-", NexusPagePrefix + "v1-"}

// RetiredCacheSuffixes end the names of files no producer writes any more: GitHub's unversioned release lists and
// release notes. No current name ends this way (their successors end -v2.json).
var RetiredCacheSuffixes = []string{"-notes.json", "-releases.json"}

// Retired reports whether the cache file rel (slash form, below the cache folder) is one no producer writes any more.
func Retired(rel string) bool {
	if strings.HasPrefix(rel, "nexus/mod-") && !strings.HasPrefix(rel, NexusModPrefix) {
		return true
	}
	for _, p := range RetiredCachePrefixes {
		if strings.HasPrefix(rel, p) {
			return true
		}
	}
	for _, s := range RetiredCacheSuffixes {
		if strings.HasSuffix(rel, s) {
			return true
		}
	}
	return false
}
