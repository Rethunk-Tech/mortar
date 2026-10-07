package meta

// Name prefixes of the cache files that expire by age. Producers build their file names from these and Clean up
// reads its expiry rules by them, so a producer's files are always claimed by a rule.
const (
	NexusDetailsPrefix    = "nexus/details-"
	NexusPagePrefix       = "nexus/page-"
	NexusCategoriesPrefix = "nexus/categories-"
	DatasetPrefix         = "dataset-"
)

// RetiredCachePrefixes name files no producer writes any more; Clean up removes them whatever their age.
var RetiredCachePrefixes = []string{"nexus-requirements-", NexusPagePrefix + "v1-"}
