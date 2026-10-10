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

// Nexus's logo is artwork rather than a vector mark, so SourceLogo draws it apart from ICONS.
const hasSourceLogo = (id: string) => id === 'nexus' || id in ICONS

export { hasSourceLogo, ICONS }
