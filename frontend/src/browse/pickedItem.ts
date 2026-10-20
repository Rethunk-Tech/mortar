import type { BrowseItem } from './browseTypes.ts'

// The card's hit as the chosen source sees it: the same mod on another source carries that source's own id, page,
// installed state and flags.
function pickedItem(item: BrowseItem, source: string): BrowseItem {
  const alt = item.alts?.find((a) => a.source === source)
  if (!alt || source === item.source) {
    return item
  }
  return {
    ...item,
    source: alt.source,
    id: alt.id,
    url: alt.url,
    installed: alt.installed,
    obsolete: alt.obsolete,
    broken: alt.broken,
    endorsements: alt.endorsements,
    stars: alt.stars,
    downloads: alt.downloads,
    version: alt.version,
    updated: alt.updated,
    size: alt.size,
    bundled: false,
  }
}

export { pickedItem }
