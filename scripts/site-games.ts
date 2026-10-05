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

function listed(names: string[]): string {
  if (names.length <= 1) {
    return names.join('')
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`
}

const games = (JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: Game[] }).games
const text = listed(games.filter((g) => g.enabled).map((g) => g.name))
const check = process.argv.includes('--check')
let stale = false
for (const page of PAGES) {
  const before = readFileSync(page, 'utf8')
  const after = before.replace(BLOCK, `$1${text}$2`)
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
