import { i18n } from '@lingui/core'
import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import type { ListRow } from '../mods/listColumns.ts'
import { formatCount } from '../mods/nexusFormat.ts'
import { GITHUB, NEXUS, THUNDERSTORE } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'

const MODRINTH = 'modrinth'
const VSMODDB = 'vsmoddb'
const PARADOX = 'paradox'
const MODIO = 'modio'
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

/** A hit's popularity, which each site counts differently: Nexus endorsements, Thunderstore and Paradox Mods likes,
 * Modrinth and Mod DB follows, and mod.io positive ratings all land in endorsements, and only GitHub has stars. */
function usePopularity(item: BrowseItem): string {
  const { endorsements, stars } = item
  switch (item.source) {
    case NEXUS:
      return plural({ count: endorsements }, { one: '# endorsement', other: '# endorsements' })
    case GITHUB:
      return plural({ count: stars }, { one: '# star', other: '# stars' })
    case THUNDERSTORE:
    case PARADOX:
      return plural({ count: endorsements }, { one: '# like', other: '# likes' })
    case MODRINTH:
    case VSMODDB:
      return plural({ count: endorsements }, { one: '# follower', other: '# followers' })
    case MODIO:
      return plural(endorsements, { one: '# positive rating', other: '# positive ratings' })
    default:
      return ''
  }
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

const POPULARITY_ICON: Record<string, StatIcon> = {
  [NEXUS]: 'endorsement',
  [GITHUB]: 'star',
  [THUNDERSTORE]: 'like',
  [PARADOX]: 'like',
  [MODRINTH]: 'follower',
  [VSMODDB]: 'follower',
}

const COMPACT_FROM = 1000
const compact = (n: number) => formatCount(n, i18n.locale || 'en')

/** A card's chosen stats, leaving out what the hit's site does not say. */
function useCardStats(item: BrowseItem, stats: readonly CardStat[]): CardStatPart[] {
  const { t } = useLingui()
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
            icon: POPULARITY_ICON[item.source] ?? 'like',
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
    updated:
      updated === ''
        ? null
        : { id: 'updated', icon: 'updated', short: updated, full: t`Updated ${{ when: updated }}` },
  }
  return stats.flatMap((s) => parts[s] ?? [])
}

/** An installed mod's card line for the same chosen stats: its Nexus page's figures and its size on disk. The card
 * already shows the installed version. `short` counts compactly to fit the card; `full` is exact, for the tooltip. */
function useModCardStats(
  { mod, size, details }: Pick<ListRow, 'mod' | 'size' | 'details'>,
  stats: readonly CardStat[],
): { short: string[]; full: string[] } {
  const { t } = useLingui()
  const page = details?.page
  const endorsements = page?.endorsements ?? mod.endorsements
  const updated = page?.updated ? t`Updated ${{ when: formatWhen(page.updated) }}` : ''
  const sized = size !== undefined && size > 0 ? formatBytes(size) : ''
  const endorsementsFull =
    endorsements > 0
      ? plural({ count: endorsements }, { one: '# endorsement', other: '# endorsements' })
      : ''
  const downloadsFull = page
    ? plural({ count: page.downloads }, { one: '# download', other: '# downloads' })
    : ''
  // Below 1,000 the compact figure is the exact one, so the plural form serves.
  const endorsementsShort =
    endorsements >= COMPACT_FROM
      ? t`${{ count: compact(endorsements) }} endorsements`
      : endorsementsFull
  const downloadsShort =
    page && page.downloads >= COMPACT_FROM
      ? t`${{ count: compact(page.downloads) }} downloads`
      : downloadsFull
  const pick = (parts: Record<CardStat, string>) =>
    stats.map((s) => parts[s]).filter((p) => p !== '')
  return {
    short: pick({
      popularity: endorsementsShort,
      downloads: downloadsShort,
      size: sized,
      version: '',
      updated,
    }),
    full: pick({
      popularity: endorsementsFull,
      downloads: downloadsFull,
      size: sized,
      version: '',
      updated,
    }),
  }
}

export type { CardStat, CardStatPart, StatIcon }
export { CARD_STATS, formatCardStats, parseCardStats, useCardStats, useModCardStats, useStats }
