// Writes every generated part of site/ so nothing there is kept by hand twice: the nav, footer and search metadata of
// the hand-written pages, the landing page's game list and count, a page and a share page per playable game, the
// changelog, the user guide, the 404 page, the Discord redirects under /go/, robots.txt and sitemap.xml.
// `--check` fails instead of writing when anything is stale; the gate runs it that way.
// Either way it fails when a picture under site/ is linked from no page, stylesheet or script, so none ships unused.
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, dirname } from 'node:path'
import { refreshSteamArt, steamArt } from './site/art.ts'
import { brandArt, brandSprite } from './site/brands.ts'
import { changelogFeed, changelogPage } from './site/changelog.ts'
import { footer, nav, type Page, page, SITE, seo, urlOf } from './site/chrome.ts'
import {
  appVersion,
  docPage,
  faqPage,
  landingFaq,
  refreshRivalFacts,
  whyPage,
  whyPages,
} from './site/faq.ts'
import { gamePage, gamesBand, gamesIndexPage, playableGames, sharePage } from './site/games.ts'
import { guidePage } from './site/guide.ts'

const block = (name: string) =>
  new RegExp(`(<!-- ${name}:start -->)[\\s\\S]*?(<!-- ${name}:end -->)`, 'g')

const PAGES: Page[] = [
  { path: 'site/index.html', root: '', footer: { agpl: true } },
  {
    path: 'site/download/index.html',
    root: '../',
    current: 'download',
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

function unescapeHTML(raw: string): string {
  return raw
    .replace(/&#(\d+);/g, (_, n: string) => String.fromCharCode(Number(n)))
    .replaceAll('&quot;', '"')
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&amp;', '&')
}

function appLd(): object {
  const version = appVersion()
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Mortar',
    applicationCategory: 'GameApplication',
    operatingSystem: 'Windows, Linux',
    softwareVersion: version,
    url: `${SITE}/`,
    downloadUrl: `${SITE}/download/`,
    license: 'https://www.gnu.org/licenses/agpl-3.0.html',
    offers: { '@type': 'Offer', price: '0', priceCurrency: 'USD' },
  }
}

/** A hand-written page with its nav, footer, metadata and (on the landing page) game list filled in. */
function handPage(
  p: Page,
  playable: ReturnType<typeof playableGames>,
  steam: ReturnType<typeof steamArt>,
): string {
  const before = readFileSync(p.path, 'utf8')
  // Function replacements, so `$&` in a game name is not read as a replacement pattern.
  const fill = (src: string, name: string, body: string) =>
    src.replace(block(name), (_, open: string, close: string) => `${open}${body}${close}`)
  const chrome = (html: string) => `\n    ${html}\n    `
  const outside = before.replace(block('seo'), '')
  const title = /<title>([^<]*)<\/title>/.exec(outside)?.[1]
  const description = /<meta\s+name="description"\s+content="([^"]*)"/.exec(outside)?.[1]
  if (title === undefined || description === undefined) {
    throw new Error(`site: ${p.path} needs a <title> and a meta description`)
  }
  let after = fill(before, 'nav', chrome(nav(p)))
  after = fill(after, 'foot', chrome(footer(p)))
  const landing = p.path === 'site/index.html'
  if (landing) {
    after = fill(after, 'faq', landingFaq(playable.length, whyPages()))
    after = fill(after, 'game-count', String(playable.length))
    after = fill(after, 'games-band', gamesBand(playable, steam))
  }
  after = fill(
    after,
    'seo',
    chrome(
      seo({
        url: urlOf(p.path),
        title: unescapeHTML(title),
        description: unescapeHTML(description),
        ownTitle: /property="og:title"/.test(outside),
        ownDescription: /property="og:description"/.test(outside),
        ...(landing ? { jsonLd: appLd(), feed: true } : {}),
      }),
    ),
  )
  return after
}

function notFoundPage(): string {
  return page({
    path: 'site/404.html',
    title: 'Page not found: Mortar',
    description: 'This page does not exist.',
    index: false,
    body: `      <h1 class="ext-title">Page not found</h1>
      <p class="lede">Nothing lives at this address. A shared profile link opens at <code>mortar.rethunk.tech/&lt;game&gt;/p</code>; check the link was copied whole.</p>
      <div class="cta ext-cta">
        <a class="btn" href="/">Mortar home</a>
        <a class="btn-ghost" href="/guide/">User guide</a>
      </div>`,
  })
}

// Permanent invites, by the route the app opens; a moved channel needs only a change here.
const INVITES: Record<string, string> = {
  'bug-reports': 'https://discord.gg/9hKFwB8xGW',
  help: 'https://discord.gg/4ZNKgGwYdw',
  suggestions: 'https://discord.gg/sgp8MHXpH8',
}

/** A page the app opens that forwards to a Discord invite: unlisted, unindexed, with a link for browsers that stay. */
function redirectPage(invite: string): string {
  return `<!doctype html>
<!-- Written by scripts/site.ts; edit the generator, not this file. -->
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="color-scheme" content="dark" />
    <meta name="robots" content="noindex" />
    <meta http-equiv="refresh" content="0; url=${invite}" />
    <link rel="canonical" href="${invite}" />
    <title>Opening Mortar's Discord</title>
    <link rel="icon" href="/favicon.png" />
    <link rel="stylesheet" href="/base.css" />
  </head>
  <body>
    <main class="wrap">
      <p><a href="${invite}">Open Mortar's Discord</a></p>
    </main>
  </body>
</html>
`
}

const games = playableGames()
if (process.argv.includes('--refresh')) {
  await refreshRivalFacts()
  await refreshSteamArt(
    games.flatMap((g) => (g.stores?.steam as { appId?: string } | undefined)?.appId ?? []),
  )
}
const art = steamArt()
const missingArt = games.filter((g) => {
  const appId = (g.stores?.steam as { appId?: string } | undefined)?.appId
  return appId !== undefined && art[appId] === undefined
})
if (missingArt.length > 0) {
  console.error(
    `site: no Steam art address for ${missingArt.map((g) => g.id).join(', ')}: run bun scripts/site.ts --refresh`,
  )
  process.exit(1)
}
const files = new Map<string, string | Buffer>()
files.set('site/games/index.html', gamesIndexPage(games, art))
for (const g of games) {
  files.set(`site/${g.id}/index.html`, gamePage(g, art, games))
  files.set(`site/${g.id}/p/index.html`, sharePage(g, art))
}
for (const p of PAGES) {
  files.set(p.path, handPage(p, games, art))
}
files.set('site/changelog/index.html', changelogPage())
files.set('site/changelog/feed.xml', changelogFeed())
files.set('site/img/brands.svg', brandSprite())
for (const [path, bytes] of brandArt()) {
  files.set(path, bytes)
}
files.set('site/guide/index.html', guidePage())
const whys = whyPages()
const counts = {
  games: games.length,
  thunderstore: games.filter((g) => g.sources?.some((s) => s.id === 'thunderstore')).length,
}
files.set('site/faq/index.html', faqPage(games.length, whys))
for (const w of whys) {
  files.set(`site/why/${w.id}/index.html`, whyPage(w, counts))
}
const botDocs = [
  ['terms', 'The terms for the Mortar bot that runs the Mortar Mod Manager Discord server.'],
  [
    'privacy',
    'What the Mortar bot in the Mortar Mod Manager Discord server reads and keeps, and why.',
  ],
]
for (const [doc, description] of botDocs) {
  files.set(
    `site/bot/${doc}/index.html`,
    docPage(`scripts/site/bot/${doc}.md`, `site/bot/${doc}/index.html`, description),
  )
}
files.set('site/404.html', notFoundPage())
for (const [route, invite] of Object.entries(INVITES)) {
  files.set(`site/go/${route}/index.html`, redirectPage(invite))
}
const listedPages = [
  ...PAGES.map((p) => p.path),
  'site/games/index.html',
  'site/faq/index.html',
  ...botDocs.map(([doc]) => `site/bot/${doc}/index.html`),
  ...whys.map((w) => `site/why/${w.id}/index.html`),
  'site/guide/index.html',
  'site/changelog/index.html',
  ...games.map((g) => `site/${g.id}/index.html`),
]
files.set(
  'site/sitemap.xml',
  `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${listedPages.map((p) => `  <url><loc>${urlOf(p)}</loc></url>`).join('\n')}
</urlset>
`,
)
files.set('site/robots.txt', `User-agent: *\nAllow: /\n\nSitemap: ${SITE}/sitemap.xml\n`)

const check = process.argv.includes('--check')
const stale = [...files].filter(
  ([path, want]) =>
    !existsSync(path) ||
    (typeof want === 'string'
      ? readFileSync(path, 'utf8') !== want
      : !readFileSync(path).equals(want)),
)
if (check && stale.length > 0) {
  console.error(
    `site pages are stale (${stale.map(([p]) => p).join(', ')}): run bun scripts/site.ts`,
  )
  process.exit(1)
}
for (const [path, want] of stale) {
  mkdirSync(dirname(path), { recursive: true })
  writeFileSync(path, want)
}

const PICTURE = /\.(webp|png|jpe?g|svg|ico|avif|gif)$/
const siteFiles = readdirSync('site', { recursive: true, encoding: 'utf8' })
const linking = [
  ...siteFiles
    .filter((f) => /\.(html|css|js)$/.test(f))
    .map((f) => readFileSync(`site/${f}`, 'utf8')),
  readFileSync('scripts/site/chrome.ts', 'utf8'),
].join('\n')
const unlinked = siteFiles.filter((f) => PICTURE.test(f) && !linking.includes(basename(f)))
if (unlinked.length > 0) {
  console.error(`site pictures nothing links: ${unlinked.join(', ')}`)
  process.exit(1)
}
