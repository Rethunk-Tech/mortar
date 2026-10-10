// Writes the playable games from the bundled catalog into the site's marked blocks, the nav and footer into the
// pages that share them, and each game's share page (site/<game>/p/, where its share links point) from
// scripts/share-page.html, so the site never lists games by hand or copies its chrome.
// `--check` fails instead of writing when anything is stale; the gate runs it that way.
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'

const CATALOG = 'internal/components/components.json'
const REPO = 'https://github.com/Rethunk-Tech/mortar'
const DISCORD = 'https://discord.gg/TfecaTm8J2'
const SHARE_PAGE = 'scripts/share-page.html'
const block = (name: string) =>
  new RegExp(`(<!-- ${name}:start -->)[\\s\\S]*?(<!-- ${name}:end -->)`, 'g')

// What differs between pages: where the site root is from the page, which nav link is the current page, whether the
// nav links the FAQ, and which extra links the footer carries.
interface Page {
  path: string
  root: string
  current?: 'download' | 'extension'
  faq?: boolean
  footer: { downloads?: boolean; agpl?: boolean; source?: string }
}

const PAGES: Page[] = [
  { path: 'site/index.html', root: '', faq: true, footer: { downloads: true, agpl: true } },
  {
    path: 'site/download/index.html',
    root: '../',
    current: 'download',
    faq: true,
    footer: { agpl: true },
  },
  { path: 'site/code-signing/index.html', root: '../', footer: {} },
  { path: 'site/privacy/index.html', root: '../', footer: {} },
  {
    path: 'site/extension/index.html',
    root: '../',
    current: 'extension',
    footer: { agpl: true, source: 'https://github.com/Rethunk-Tech/mortar-browser-extension' },
  },
]

function nav(p: Page): string {
  const home = p.root ? '/' : ''
  const link = (href: string, label: string, optional: boolean, current?: string) =>
    `<a${optional ? ' class="optional"' : ''} href="${href}"${current && p.current === current ? ' aria-current="page"' : ''}>${label}</a>`
  return `<nav class="nav" aria-label="Main">
      <div class="wrap">
        <a class="brand" href="/">
          <svg width="28" height="28" aria-hidden="true"><use href="${p.root}img/icons.svg#brand" /></svg>
          Mortar
        </a>
        <div class="nav-links">
          ${link(`${home}#features`, 'Features', true)}
          ${link('/extension/', 'Extension', true, 'extension')}
          ${p.faq ? `${link(`${home}#faq`, 'FAQ', true)}\n          ` : ''}${link('/download/', 'Download', false, 'download')}
          <a href="${REPO}">GitHub</a>
          <a class="optional" href="${DISCORD}">Discord</a>
        </div>
      </div>
    </nav>`
}

function footer(p: Page): string {
  const { downloads, agpl, source = REPO } = p.footer
  const links = [
    '<a href="/privacy/">Privacy</a>',
    '<a href="/code-signing/">Code signing</a>',
    `<a href="${source}">Source</a>`,
    `<a href="${DISCORD}">Discord</a>`,
    ...(downloads ? ['<a href="/download/">Downloads</a>'] : []),
    ...(agpl ? ['<a href="https://www.gnu.org/licenses/agpl-3.0.html">AGPL-3.0</a>'] : []),
  ]
  return `<footer class="foot">
      <div class="wrap">
        <span>Mortar by <a href="https://we.rethunk.tech/">Rethunk.Tech</a>. Not affiliated with any game developer or mod site.</span>
        <span>${links.join(' · ')}</span>
      </div>
    </footer>`
}

interface Game {
  id: string
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

const { games } = JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: Game[] }
const playable = games.filter((g) => g.enabled)
const text = escapeHTML(listed(playable.map((g) => g.name)))
const check = process.argv.includes('--check')
let stale = false
const sharePage = readFileSync(SHARE_PAGE, 'utf8')
for (const g of playable) {
  const path = `site/${g.id}/p/index.html`
  const want = sharePage.replaceAll('GAME_NAME', escapeHTML(g.name))
  if (!existsSync(path) || readFileSync(path, 'utf8') !== want) {
    stale = true
    if (!check) {
      mkdirSync(`site/${g.id}/p`, { recursive: true })
      writeFileSync(path, want)
    }
  }
}
for (const page of PAGES) {
  const before = readFileSync(page.path, 'utf8')
  // Function replacements, so `$&` in a game name is not read as a replacement pattern.
  const fill = (src: string, name: string, body: string) =>
    src.replace(block(name), (_, open: string, close: string) => `${open}${body}${close}`)
  const chrome = (html: string) => `\n    ${html}\n    `
  let after = fill(before, 'nav', chrome(nav(page)))
  after = fill(after, 'foot', chrome(footer(page)))
  if (page.path === 'site/index.html') {
    after = fill(after, 'games', text)
  }
  if (after !== before) {
    stale = true
    if (!check) {
      writeFileSync(page.path, after)
    }
  }
}
if (check && stale) {
  console.error('site game lists or share pages are stale: run bun scripts/site-games.ts')
  process.exit(1)
}
