import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { GITHUB, NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'

const MODRINTH = 'modrinth'

/** A hit's popularity line. Each site counts a different thing into the same fields: Nexus endorsements, a
 * Thunderstore rating and Modrinth follows all land in endorsements, and only GitHub has stars. */
function useStats(item: BrowseItem): string {
  const { t } = useLingui()
  const { endorsements, stars, downloads } = item
  const dl = plural(downloads, { one: '# download', other: '# downloads' })
  switch (item.source) {
    case NEXUS:
      return t`${plural(endorsements, { one: '# endorsement', other: '# endorsements' })} · ${dl}`
    case GITHUB:
      return plural(stars, { one: '# star', other: '# stars' })
    case THUNDERSTORE:
      return t`${plural(endorsements, { one: '# like', other: '# likes' })} · ${dl}`
    case MODRINTH:
      return t`${plural(endorsements, { one: '# follower', other: '# followers' })} · ${dl}`
    default:
      return dl
  }
}

export { useStats }
