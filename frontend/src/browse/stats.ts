import { i18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import type { ListRow } from '../mods/listColumns.ts'
import { formatCount } from '../mods/nexusFormat.ts'
import { space } from '../theme/density.ts'
import { GITHUB, NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'

const MODRINTH = 'modrinth'
const VSMODDB = 'vsmoddb'
const PARADOX = 'paradox'
const MODIO = 'modio'
const CURSEFORGE = 'curseforge'
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

/** Which mark a stat shows; popularity's depends on what the hit's site counts. */
type StatIcon =
  | 'like'
  | 'endorsement'
  | 'star'
  | 'follower'
  | 'download'
  | 'size'
  | 'version'
  | 'updated'

/** How a source counts popularity: card icon, count copy, and the endorsements-sort label. */
type PopularityKind = 'endorsement' | 'like' | 'star' | 'follower' | 'rating' | 'popular'

const POPULARITY: Record<string, { icon: StatIcon; kind: PopularityKind }> = {
  [NEXUS]: { icon: 'endorsement', kind: 'endorsement' },
  [GITHUB]: { icon: 'star', kind: 'star' },
  [THUNDERSTORE]: { icon: 'like', kind: 'like' },
  [PARADOX]: { icon: 'like', kind: 'like' },
  [MODRINTH]: { icon: 'follower', kind: 'follower' },
  [VSMODDB]: { icon: 'follower', kind: 'follower' },
  [MODIO]: { icon: 'like', kind: 'rating' },
  [CURSEFORGE]: { icon: 'like', kind: 'popular' },
}

function popularityIcon(source: string): StatIcon {
  return POPULARITY[source]?.icon ?? 'like'
}

function popularityWording(source: string, count: number): string {
  switch (POPULARITY[source]?.kind) {
    case 'endorsement':
      return plural({ count }, { one: '# endorsement', other: '# endorsements' })
    case 'like':
      return plural({ count }, { one: '# like', other: '# likes' })
    case 'star':
      return plural({ count }, { one: '# star', other: '# stars' })
    case 'follower':
      return plural({ count }, { one: '# follower', other: '# followers' })
    case 'rating':
      return plural(count, { one: '# positive rating', other: '# positive ratings' })
    default:
      return ''
  }
}

function popularitySortLabel(source: string): string {
  switch (POPULARITY[source]?.kind) {
    case 'follower':
      return i18n._(msg`Sort: most followed`)
    case 'like':
    case 'rating':
      return i18n._(msg`Sort: top rated`)
    case 'popular':
      return i18n._(msg`Sort: most popular`)
    default:
      return i18n._(msg`Sort: most endorsed`)
  }
}

/** A hit's popularity, which each site counts differently: Nexus endorsements, Thunderstore and Paradox Mods likes,
 * Modrinth and Mod DB follows, and mod.io positive ratings all land in endorsements, and only GitHub has stars. */
function usePopularity(item: BrowseItem): string {
  const count = item.source === GITHUB ? item.stars : item.endorsements
  return popularityWording(item.source, count)
}

/** A hit's popularity line for its details: popularity and downloads (GitHub counts no downloads). */
function useStats(item: BrowseItem): string {
  const popularity = usePopularity(item)
  const dl = plural({ count: item.downloads }, { one: '# download', other: '# downloads' })
  if (item.source === GITHUB) {
    return popularity
  }
  return popularity === '' ? dl : `${popularity} · ${dl}`
}

/** One stat on a Browse card: an icon, a short figure, and the whole wording for its tooltip and screen readers. */
interface CardStatPart {
  id: CardStat
  icon: StatIcon
  short: string
  full: string
}

const compact = (n: number) => formatCount(n, i18n.locale || 'en')

/** How one stat sits on a card; each card lays out the line that holds them. */
const CARD_STAT_SX = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 0.375,
  whiteSpace: 'nowrap',
  '&:not(:last-child)': { mr: space.gap },
}

/** The "updated" stat for an already formatted date, or null when the site gives none. */
function useUpdatedPart(): (updated: string) => CardStatPart | null {
  const { t } = useLingui()
  return (updated) =>
    updated === ''
      ? null
      : { id: 'updated', icon: 'updated', short: updated, full: t`Updated ${{ when: updated }}` }
}

/** A card's chosen stats, leaving out what the hit's site does not say. */
function useCardStats(item: BrowseItem, stats: readonly CardStat[]): CardStatPart[] {
  const { t } = useLingui()
  const updatedPart = useUpdatedPart()
  const popularity = usePopularity(item)
  const version = item.version.replace(leadingV, '')
  const size = item.size > 0 ? formatBytes(item.size) : ''
  const updated = item.updated === '' ? '' : formatWhen(item.updated)
  const parts: Record<CardStat, CardStatPart | null> = {
    popularity:
      popularity === ''
        ? null
        : {
            id: 'popularity',
            icon: popularityIcon(item.source),
            short: compact(item.source === GITHUB ? item.stars : item.endorsements),
            full: popularity,
          },
    downloads:
      item.source === GITHUB
        ? null
        : {
            id: 'downloads',
            icon: 'download',
            short: compact(item.downloads),
            full: plural({ count: item.downloads }, { one: '# download', other: '# downloads' }),
          },
    size: size === '' ? null : { id: 'size', icon: 'size', short: size, full: size },
    version:
      version === ''
        ? null
        : { id: 'version', icon: 'version', short: version, full: t`Version ${{ version }}` },
    updated: updatedPart(updated),
  }
  return stats.flatMap((s) => parts[s] ?? [])
}

/** An installed mod's card line for the same chosen stats: its Nexus page's figures and its size on disk. The card
 * already shows the installed version. */
function useModCardStats(
  { mod, size, details, source }: Pick<ListRow, 'mod' | 'size' | 'details'> & { source: string },
  stats: readonly CardStat[],
): CardStatPart[] {
  const updatedPart = useUpdatedPart()
  const page = details?.page
  const endorsements = page?.endorsements ?? mod.endorsements
  const popularity = popularityWording(source, endorsements)
  const updated = page?.updated ? formatWhen(page.updated) : ''
  const sized = size !== undefined && size > 0 ? formatBytes(size) : ''
  const parts: Record<CardStat, CardStatPart | null> = {
    popularity:
      endorsements > 0 && popularity !== ''
        ? {
            id: 'popularity',
            icon: popularityIcon(source),
            short: compact(endorsements),
            full: popularity,
          }
        : null,
    downloads: page
      ? {
          id: 'downloads',
          icon: 'download',
          short: compact(page.downloads),
          full: plural({ count: page.downloads }, { one: '# download', other: '# downloads' }),
        }
      : null,
    size: sized === '' ? null : { id: 'size', icon: 'size', short: sized, full: sized },
    version: null,
    updated: updatedPart(updated),
  }
  return stats.flatMap((s) => parts[s] ?? [])
}

export type { CardStat, CardStatPart, StatIcon }
export {
  CARD_STAT_SX,
  CARD_STATS,
  formatCardStats,
  parseCardStats,
  popularitySortLabel,
  useCardStats,
  useModCardStats,
  useStats,
}
