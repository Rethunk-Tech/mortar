// Addresses and helpers every page shares. Build-time code reads repo files through repo(), which finds the repo root
// from wherever the build was started.
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import process from 'node:process'

export const SITE = 'https://mortar.rethunk.tech'
export const REPO = 'https://github.com/Rethunk-Tech/mortar'
export const DISCORD = 'https://discord.gg/TfecaTm8J2'
// The Mortar packages OpenPGP key, which signs the package repositories and each release's SHA256SUMS.
export const PACKAGES_FINGERPRINT = 'D8B1 C4C2 05C5 FB36 33CB 8B99 00B7 5959 B637 DCB5'

// Permanent invites, by the route the app opens under /go/; a moved channel needs only a change here.
export const INVITES: Record<string, string> = {
  'bug-reports': 'https://discord.gg/9hKFwB8xGW',
  help: 'https://discord.gg/4ZNKgGwYdw',
  suggestions: 'https://discord.gg/sgp8MHXpH8',
}

function rootFrom(dir: string): string {
  if (existsSync(join(dir, 'go.mod'))) {
    return dir
  }
  const up = dirname(dir)
  if (up === dir) {
    throw new Error('site: no go.mod above the working directory')
  }
  return rootFrom(up)
}

const ROOT = rootFrom(process.cwd())

/** A path in the repo, from its root. */
export const repo = (path: string) => join(ROOT, path)

export function escapeHTML(raw: string): string {
  return raw.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)
}

export function listed(names: string[]): string {
  if (names.length <= 1) {
    return names.join('')
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`
}

/** Markdown as HTML, with heading ids and tables wrapped so they scroll on a narrow screen. */
export function markdown(md: string): string {
  return Bun.markdown
    .html(md, { headings: { ids: true } })
    .replaceAll('<table>', '<div class="table-wrap"><table>')
    .replaceAll('</table>', '</table></div>')
}
