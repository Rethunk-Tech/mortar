import {
  siCurseforge,
  siGithub,
  siItchdotio,
  siModrinth,
  siPatreon,
  siThunderstore,
} from 'simple-icons'

// GitHub's and Patreon's brand marks are black, so they use the ink colour (their own dark-mode treatment); the
// others keep their colour.
const ICONS: Record<string, { path: string; fill: string }> = {
  github: { path: siGithub.path, fill: 'var(--mortar-ink)' },
  thunderstore: { path: siThunderstore.path, fill: `#${siThunderstore.hex}` },
  modrinth: { path: siModrinth.path, fill: `#${siModrinth.hex}` },
  curseforge: { path: siCurseforge.path, fill: `#${siCurseforge.hex}` },
  itch: { path: siItchdotio.path, fill: `#${siItchdotio.hex}` },
  patreon: { path: siPatreon.path, fill: 'var(--mortar-ink)' },
}

// Nexus's and the Vintage Story Mod DB's logos are artwork rather than vector marks, so SourceLogo draws them apart
// from ICONS.
const hasSourceLogo = (id: string) => id === 'nexus' || id === 'vsmoddb' || id in ICONS

// A mark laid over a corner of a mod's picture (its source, the favourite star) on a solid chip, readable on any art.
const pictureMarkSx = {
  display: 'grid',
  placeItems: 'center',
  p: '1px',
  borderRadius: '4px',
  bgcolor: 'var(--mortar-raised)',
  border: '1px solid var(--mortar-hairline)',
} as const

// Every mark on a mod's picture is this size, so a source logo and the favourite star match wherever they show.
const PICTURE_MARK_PX = 16

export { hasSourceLogo, ICONS, PICTURE_MARK_PX, pictureMarkSx }
