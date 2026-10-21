import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import type { ListRow } from '../mods/listColumns.ts'
import { GITHUB, NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'

const MODRINTH = 'modrinth'
const VSMODDB = 'vsmoddb'
const PARADOX = 'paradox'
const leadingV = /^v(?=\d)/i

/** The stats a Browse card can show under its summary, in the order it shows them (the browseCardStats setting). */
const CARD_STATS = ['popularity', 'downloads', 'size', 'version', 'updated'] as const
type CardStat = (typeof CARD_STATS)[number]

/** The browseCardStats setting as ids in display order; "none" is an empty list. */
function parseCardStats(saved: string): CardStat[] {
  const ids = saved.split(',')
  return CARD_STATS.filter((s) => ids.includes(s))
}

function formatCardStats(stats: readonly CardStat[]): string {
  const ordered = CARD_STATS.filter((s) => stats.includes(s))
  return ordered.length === 0 ? 'none' : ordered.join(',')
}

/** A hit's popularity, which each site counts differently: Nexus endorsements and Paradox ratings, a Thunderstore rating, and Modrinth
 * and Mod DB follows all land in endorsements, and only GitHub has stars. */
function usePopularity(item: BrowseItem): string {
  const { endorsements, stars } = item
  switch (item.source) {
    case NEXUS:
    case PARADOX:
      return plural(endorsements, { one: '# endorsement', other: '# endorsements' })
    case GITHUB:
      return plural(stars, { one: '# star', other: '# stars' })
    case THUNDERSTORE:
      return plural(endorsements, { one: '# like', other: '# likes' })
    case MODRINTH:
    case VSMODDB:
      return plural(endorsements, { one: '# follower', other: '# followers' })
    default:
      return ''
  }
}

/** A hit's popularity line for its details: popularity and downloads (GitHub counts no downloads). */
function useStats(item: BrowseItem): string {
  const { t } = useLingui()
  const popularity = usePopularity(item)
  const dl = plural(item.downloads, { one: '# download', other: '# downloads' })
  if (item.source === GITHUB) {
    return popularity
  }
  return popularity === '' ? dl : t`${popularity} · ${dl}`
}

/** The parts of a card's stats line for the chosen stats, leaving out what the hit's site does not say. */
function useCardStats(item: BrowseItem, stats: readonly CardStat[]): string[] {
  const { t } = useLingui()
  const popularity = usePopularity(item)
  const parts: Record<CardStat, string> = {
    popularity,
    downloads:
      item.source === GITHUB
        ? ''
        : plural(item.downloads, { one: '# download', other: '# downloads' }),
    size: item.size > 0 ? formatBytes(item.size) : '',
    version: item.version === '' ? '' : t`Version ${item.version.replace(leadingV, '')}`,
    updated: item.updated === '' ? '' : t`Updated ${formatWhen(item.updated)}`,
  }
  return stats.map((s) => parts[s]).filter((p) => p !== '')
}

/** An installed mod's card line for the same chosen stats: its Nexus page's figures and its size on disk. The card
 * already shows the installed version. */
function useModCardStats(row: ListRow, stats: readonly CardStat[]): string[] {
  const { t } = useLingui()
  const page = row.details?.page
  const endorsements = page?.endorsements ?? row.mod.endorsements
  const parts: Record<CardStat, string> = {
    popularity:
      endorsements > 0
        ? plural(endorsements, { one: '# endorsement', other: '# endorsements' })
        : '',
    downloads: page ? plural(page.downloads, { one: '# download', other: '# downloads' }) : '',
    size: row.size !== undefined && row.size > 0 ? formatBytes(row.size) : '',
    version: '',
    updated: page?.updated ? t`Updated ${formatWhen(page.updated)}` : '',
  }
  return stats.map((s) => parts[s]).filter((p) => p !== '')
}

export type { CardStat }
export { CARD_STATS, formatCardStats, parseCardStats, useCardStats, useModCardStats, useStats }
