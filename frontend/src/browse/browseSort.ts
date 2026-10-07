import { ALL } from './browseConstants.ts'
import type { BrowseSource } from './browseTypes.ts'

// What All sources can order by: the sorts every source that has them honours on its own server and whose values
// compare across sources. Mirrors browse.MergedSorts in Go.
const MERGED_SORTS = ['downloads', 'updated']

// sortsFor lists the sorts the selected source orders by (best match, '', is always offered).
// A fixed empty list: the query effect depends on the filter built from it, so a fresh array each render would
// re-run the search.
const NONE: string[] = []

function sortsFor(source: string, sources: BrowseSource[]): string[] {
  if (source === ALL) {
    return MERGED_SORTS
  }
  return sources.find((s) => s.id === source)?.sorts ?? NONE
}

// effectiveSort falls back to best match when the stored sort is not one the selected source honours, so choosing
// GitHub's Most stars and then Nexus never sends Nexus a sort it cannot do.
function effectiveSort(sort: string, allowed: string[]): string {
  return allowed.includes(sort) ? sort : ''
}

export { effectiveSort, MERGED_SORTS, sortsFor }
