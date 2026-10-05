// Writes the playable games from the bundled catalog into the site's marked blocks, so the site never lists games by
// hand. `--check` fails instead of writing when a block is stale; the gate runs it that way.
import { readFileSync, writeFileSync } from 'node:fs'

const CATALOG = 'internal/components/components.json'
const PAGES = ['site/index.html']
const BLOCK = /(<!-- games:start -->)[\s\S]*?(<!-- games:end -->)/g

interface Game {
  name: string
  enabled: boolean
}

function escapeHTML(raw: string): string {
  return raw.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)
}

function listed(names: string[]): string {
  if (names.length <= 1) {
    return names.join('')
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`
}

const games = (JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: Game[] }).games
const text = escapeHTML(listed(games.filter((g) => g.enabled).map((g) => g.name)))
const check = process.argv.includes('--check')
let stale = false
for (const page of PAGES) {
  const before = readFileSync(page, 'utf8')
  // A function replacement, so `$&` in a game name is not read as a replacement pattern.
  const after = before.replace(BLOCK, (_, open: string, close: string) => open + text + close)
  if (after === before) {
    continue
  }
  stale = true
  if (!check) {
    writeFileSync(page, after)
  }
}
if (check && stale) {
  console.error('site game lists are stale: run bun scripts/site-games.ts')
  process.exit(1)
}
