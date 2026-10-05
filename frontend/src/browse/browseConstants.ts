import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'

const ALL = 'all'
const NEXUS = 'nexus'
const GITHUB = 'github'
const THUNDERSTORE = 'thunderstore'
const FIRST_PAGE = 1
const PICTURE_PX = 72
const ROW_PICTURE_PX = 40
const CARD_MIN_PX = 340
const SKELETON_KEYS = ['a', 'b', 'c', 'd', 'e', 'f']
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
  if (source === GITHUB) {
    return i18n._(msg`Open on GitHub`)
  }
  if (source === THUNDERSTORE) {
    return i18n._(msg`Open on Thunderstore`)
  }
  return i18n._(msg`Open on Nexus`)
}

const list = { display: 'flex', flexDirection: 'column', gap: 0.75 } as const

const grid = {
  display: 'grid',
  gridTemplateColumns: `repeat(auto-fill, minmax(${CARD_MIN_PX}px, 1fr))`,
  gap: '6px',
} as const

export {
  ALL,
  FIRST_PAGE,
  GITHUB,
  GRAY_OPACITY,
  grid,
  ICON_SIZE,
  list,
  NEXUS,
  openPageLabel,
  PICTURE_PX,
  ROW_PICTURE_PX,
  SKELETON_KEYS,
  STALE_OPACITY,
  searchHint,
  THUNDERSTORE,
}
