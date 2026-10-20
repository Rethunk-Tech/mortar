// Mod site, store and platform marks for the game pages and the landing page's downloads, from the same simple-icons marks and vendor artwork the app draws
// (frontend/src/brand), as one SVG sprite plus copies of the artwork that is not a vector mark.
import { readFileSync } from 'node:fs'
import {
  siAppimage,
  siArchlinux,
  siCurseforge,
  siDebian,
  siDiscord,
  siEa,
  siEpicgames,
  siFedora,
  siFlatpak,
  siGithub,
  siGogdotcom,
  siHomebrew,
  siItchdotio,
  siLinux,
  siLutris,
  siModrinth,
  siPatreon,
  siSteam,
  siSteamdeck,
  siThunderstore,
} from 'simple-icons'
import { escapeHTML, repo } from './site.ts'

const VENDOR = repo('frontend/src/brand/vendor')
// The app draws Xbox as a plain X rather than Microsoft's mark (frontend/src/brand/launchers/LauncherLogo.tsx).
// A game its own installer put in place has no store, so it gets a generic install mark (Material Design's
// "download", Apache-2.0).
const INSTALL_PATH = 'M12 3v10.17l3.59-3.58L17 11l-5 5-5-5 1.41-1.41L11 13.17V3h1zM5 18h14v2H5z'
// Windows has no simple-icons mark; this is the four-pane window the site's icon sprite draws.
const WINDOWS_PATH =
  'M3 5.1 10.4 4v7.2H3V5.1Zm8.4-1.2L21 2.5v8.7h-9.6V3.9ZM3 12.4h7.4v7.2L3 18.5v-6.1Zm8.4 0H21v8.7l-9.6-1.4v-7.3Z'
// Material Design's "extension" (Apache-2.0) stands for the browser extension.
const EXTENSION_PATH =
  'M20.5 11H19V7c0-1.1-.9-2-2-2h-4V3.5C13 2.12 11.88 1 10.5 1S8 2.12 8 3.5V5H4c-1.1 0-1.99.9-1.99 2v3.8H3.5c1.49 0 2.7 1.21 2.7 2.7s-1.21 2.7-2.7 2.7H2V20c0 1.1.9 2 2 2h3.8v-1.5c0-1.49 1.21-2.7 2.7-2.7 1.49 0 2.7 1.21 2.7 2.7V22H17c1.1 0 2-.9 2-2v-4h1.5c1.38 0 2.5-1.12 2.5-2.5S21.88 11 20.5 11z'
const X_PATH = 'M4 3h4.5L12 8.6 15.5 3H20l-5.7 9L20 21h-4.5L12 15.4 8.5 21H4l5.7-9z'
const HEX_COLOUR = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i

// Marks whose brand colour is black or near it take the text colour, as the app does for GitHub and Patreon.
export const MARKS: Record<string, { path: string; colour: string }> = {
  thunderstore: { path: siThunderstore.path, colour: `#${siThunderstore.hex}` },
  curseforge: { path: siCurseforge.path, colour: `#${siCurseforge.hex}` },
  modrinth: { path: siModrinth.path, colour: `#${siModrinth.hex}` },
  github: { path: siGithub.path, colour: 'currentColor' },
  discord: { path: siDiscord.path, colour: 'currentColor' },
  itch: { path: siItchdotio.path, colour: `#${siItchdotio.hex}` },
  patreon: { path: siPatreon.path, colour: 'currentColor' },
  steam: { path: siSteam.path, colour: '#1A9FFF' },
  gog: { path: siGogdotcom.path, colour: `#${siGogdotcom.hex}` },
  epic: { path: siEpicgames.path, colour: 'currentColor' },
  ea: { path: siEa.path, colour: 'currentColor' },
  xbox: { path: X_PATH, colour: '#107C10' },
  standalone: { path: INSTALL_PATH, colour: 'currentColor' },
  lutris: { path: siLutris.path, colour: `#${siLutris.hex}` },
  flatpak: { path: siFlatpak.path, colour: `#${siFlatpak.hex}` },
  windows: { path: WINDOWS_PATH, colour: '#4CC2FF' },
  linux: { path: siLinux.path, colour: `#${siLinux.hex}` },
  steamdeck: { path: siSteamdeck.path, colour: `#${siSteamdeck.hex}` },
  appimage: { path: siAppimage.path, colour: `#${siAppimage.hex}` },
  debian: { path: siDebian.path, colour: `#${siDebian.hex}` },
  fedora: { path: siFedora.path, colour: `#${siFedora.hex}` },
  archlinux: { path: siArchlinux.path, colour: `#${siArchlinux.hex}` },
  homebrew: { path: siHomebrew.path, colour: `#${siHomebrew.hex}` },
  extension: { path: EXTENSION_PATH, colour: 'currentColor' },
}

// Artwork rather than vector marks, published from the app's copies under /img/vendor/.
export const ART: Record<string, string> = {
  nexus: 'nexusmods.svg',
  vsmoddb: 'vsmoddb.png',
  bottles: 'bottles.svg',
}

/** A #RRGGBB brand colour as rgb(), the form the site's styles use. */
export function rgb(colour: string): string {
  const hex = HEX_COLOUR.exec(colour)
  return hex
    ? `rgb(${hex
        .slice(1)
        .map((h) => Number.parseInt(h, 16))
        .join(', ')})`
    : colour
}

/** /img/brands.svg: one <symbol> per vector mark, drawn in currentColor so each use sets its colour. */
export function brandSprite(): string {
  const symbols = Object.entries(MARKS).map(
    ([id, m]) =>
      `  <symbol id="${id}" viewBox="0 0 24 24"><path fill="currentColor" d="${m.path}" /></symbol>`,
  )
  return `<svg xmlns="http://www.w3.org/2000/svg">\n  <title>Mod site and store marks</title>\n${symbols.join('\n')}\n</svg>\n`
}

/** An artwork file's bytes, by its name under /img/vendor/. */
export const brandArt = (file: string): Buffer => readFileSync(`${VENDOR}/${file}`)

// The mod sources a share link can name, by the kind the link uses.
const SHARE_SOURCES = [
  'thunderstore',
  'nexus',
  'curseforge',
  'github',
  'itch',
  'patreon',
  'vsmoddb',
]

/** How the share page badges each mod with its source: the sprite mark in its colour, or the artwork file. */
export function shareMarks(): Record<string, { colour: string } | { art: string }> {
  const out: Record<string, { colour: string } | { art: string }> = {}
  for (const id of SHARE_SOURCES) {
    const mark = MARKS[id]
    const art = ART[id]
    if (mark) {
      out[id] = { colour: rgb(mark.colour) }
    } else if (art) {
      out[id] = { art: `/img/vendor/${art}` }
    }
  }
  return out
}

/** A brand's mark as a square button, its name in the tooltip and accessible label; a link when the brand has a page
 * for the game. A brand with no mark shows its name instead. */
export function brandIcon(id: string, name: string, href?: string): string {
  const mark = MARKS[id]
  const art = ART[id]
  let body = `${escapeHTML(name)}`
  if (mark) {
    body = `<svg style="color: ${rgb(mark.colour)}" aria-hidden="true"><use href="/img/brands.svg#${id}" /></svg>`
  } else if (art) {
    body = `<img src="/img/vendor/${art}" alt="" />`
  }
  const label = escapeHTML(name)
  const hidden = mark || art ? `<span class="visually-hidden">${label}</span>` : ''
  return href
    ? `<a class="fact-icon" href="${href}" title="${label}">${body}${hidden}</a>`
    : `<span class="fact-icon" title="${label}">${body}${hidden}</span>`
}
