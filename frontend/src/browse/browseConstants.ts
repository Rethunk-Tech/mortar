import { msg } from '@lingui/core/macro'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'

const ALL = 'all'
const NEXUS = 'nexus'
const GITHUB = 'github'
const THUNDERSTORE = 'thunderstore'
const FIRST_PAGE = 1
const PICTURE_PX = 72
const ROW_PICTURE_PX = 40
const CARD_MIN_PX = 340
const GAP_PX = 6
const ICON_SIZE = 40
const STALE_OPACITY = 0.6
const GRAY_OPACITY = 0.5

function searchHint(source: string, premium: boolean): string {
  if (source === ALL) {
    return i18n._(
      msg`Search every site this game's mods come from at once. Each result installs into this profile from wherever its author published it.`,
    )
  }
  if (source === GITHUB) {
    return i18n._(
      msg`Search GitHub for mods published as releases. Add puts the latest release in this profile.`,
    )
  }
  if (source === NEXUS && premium) {
    return i18n._(msg`Search Nexus Mods. Download installs the mod into this profile.`)
  }
  return i18n._(
    msg`Search Nexus Mods. Free accounts download from the mod's page with Mod Manager Download.`,
  )
}

function openPageLabel(source: string): string {
  const label = sourceLabel(source)
  return i18n._(msg`Open on ${label}`)
}

const list = { display: 'flex', flexDirection: 'column', gap: `${GAP_PX}px` } as const

const grid = {
  display: 'grid',
  gridTemplateColumns: `repeat(auto-fill, minmax(${CARD_MIN_PX}px, 1fr))`,
  gap: `${GAP_PX}px`,
} as const

export {
  ALL,
  CARD_MIN_PX,
  FIRST_PAGE,
  GAP_PX,
  GITHUB,
  GRAY_OPACITY,
  grid,
  ICON_SIZE,
  list,
  NEXUS,
  openPageLabel,
  PICTURE_PX,
  ROW_PICTURE_PX,
  STALE_OPACITY,
  searchHint,
  THUNDERSTORE,
}
