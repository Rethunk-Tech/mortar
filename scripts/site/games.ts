// A page and a share page per playable game in the bundled catalog, and the linked game list on the landing page.
import { readFileSync } from 'node:fs'
import type { SteamArt } from './art.ts'
import { brandIcon, shareMarks } from './brands.ts'
import { escapeHTML, listed, page, SITE } from './chrome.ts'

const CATALOG = 'internal/components/components.json'

export interface Game {
  id: string
  name: string
  enabled: boolean
  // A dedicated server's profile, which shares its client's page rather than being a game of its own.
  server?: unknown
  stores?: Record<string, unknown>
  loaders?: { id: string; name: string }[]
  sources?: { id: string; key?: string }[]
}

const STORE_NAMES: Record<string, string> = {
  steam: 'Steam',
  gog: 'GOG',
  epic: 'Epic Games',
  ea: 'EA app',
  xbox: 'Xbox app',
  lutris: 'Lutris',
  bottles: 'Bottles',
  flatpak: 'Flatpak',
  standalone: 'Its own installer',
}

export const SOURCE_NAMES: Record<string, string> = {
  nexus: 'Nexus Mods',
  thunderstore: 'Thunderstore',
  curseforge: 'CurseForge',
  modrinth: 'Modrinth',
  github: 'GitHub',
  vsmoddb: 'Vintage Story Mod DB',
  itch: 'itch.io',
  patreon: 'Patreon',
}

export function playableGames(): Game[] {
  const { games } = JSON.parse(readFileSync(CATALOG, 'utf8')) as { games: Game[] }
  return games.filter((g) => g.enabled && g.server === undefined)
}

function named(ids: string[], names: Record<string, string>): string[] {
  return ids.map((id) => {
    const name = names[id]
    if (!name) {
      throw new Error(`site: no display name for "${id}"; add it to scripts/site/games.ts`)
    }
    return name
  })
}

function facts(g: Game) {
  const loaders = (g.loaders ?? []).map((l) => l.name).filter((n) => n !== '')
  return {
    loader: listed(loaders),
    stores: listed(named(Object.keys(g.stores ?? {}), STORE_NAMES)),
    sources: listed(
      named(
        (g.sources ?? []).map((s) => s.id),
        SOURCE_NAMES,
      ),
    ),
  }
}

// Steam's header and library capsule art sizes.
const HEADER = { width: 460, height: 215 }
const CAPSULE = { width: 300, height: 450 }

const steamAppId = (g: Game) => (g.stores?.steam as { appId?: string } | undefined)?.appId

/** The game's own page on a mod site, where the catalog key names one. */
function sourceURL(s: { id: string; key?: string }): string | undefined {
  switch (s.id) {
    case 'thunderstore':
      return s.key ? `https://thunderstore.io/c/${s.key}/` : undefined
    case 'nexus':
      return s.key ? `https://www.nexusmods.com/games/${s.key}/mods` : undefined
    case 'vsmoddb':
      return 'https://mods.vintagestory.at/'
    default:
      return undefined
  }
}

function gameCard(g: Game, art: Record<string, SteamArt>, data = ''): string {
  const appId = steamAppId(g)
  const a = appId ? art[appId] : undefined
  const meta = escapeHTML(facts(g).loader || 'No loader needed')
  let cover = ''
  if (a?.capsule) {
    const srcset = a.capsule2x ? ` srcset="${a.capsule} 1x, ${a.capsule2x} 2x"` : ''
    cover = `<img class="cover-art" src="${a.capsule}"${srcset} width="${CAPSULE.width}" height="${CAPSULE.height}" alt="" loading="lazy" referrerpolicy="no-referrer" />`
  } else if (a?.header) {
    cover = `<img class="cover-art" src="${a.header}" width="${HEADER.width}" height="${HEADER.height}" alt="" loading="lazy" referrerpolicy="no-referrer" />`
  }
  return `        <li${data}>
          <a class="cover${cover ? '' : ' no-art'}" href="/${g.id}/">${cover}<span class="cover-text"><span class="game-name">${escapeHTML(g.name)}</span><span class="game-meta">${meta}</span></span></a>
        </li>`
}

const MORE_GAMES = 4

/** The games after g in name order that share its loader or, for a loader only g runs, its first mod site. */
function neighbours(g: Game, games: Game[]): { title: string; games: Game[] } {
  const { loader } = facts(g)
  let same = games.filter((o) => facts(o).loader === loader)
  let title = loader ? `More games on ${escapeHTML(loader)}` : 'More games without a mod loader'
  const site = g.sources?.[0]?.id
  if (same.length < 2 && site) {
    same = games.filter((o) => o.sources?.some((s) => s.id === site))
    title = `More games with mods from ${escapeHTML(SOURCE_NAMES[site] ?? site)}`
  }
  same.sort((a, b) => a.name.localeCompare(b.name))
  const at = same.findIndex((o) => o.id === g.id)
  const count = Math.min(MORE_GAMES, same.length - 1)
  return {
    title,
    games: Array.from({ length: count }, (_, i) => same[(at + 1 + i) % same.length]).filter(
      (o): o is Game => o !== undefined,
    ),
  }
}

/** The full-bleed header of a game's pages: its Steam library art behind the breadcrumbs and the given content. */
function banner(g: Game, art: Record<string, SteamArt>, content: string): string {
  const appId = steamAppId(g)
  const heroArt = appId ? art[appId]?.hero : undefined
  return `    <header class="bleed${heroArt ? '' : ' plain'}" data-pagefind-body${heroArt ? ` style="--hero: url('${heroArt}')"` : ''}>
      <div class="wrap">
        <nav class="crumbs" aria-label="Breadcrumb" data-pagefind-ignore><a href="/">Mortar</a> › <a href="/games/">Games</a> › <a href="/${g.id}/">${escapeHTML(g.name)}</a></nav>
        ${content}
      </div>
    </header>
`
}

/** The game's own page: what Mortar does for it and how to start, from its catalog entry. */
export function gamePage(g: Game, art: Record<string, SteamArt>, games: Game[]): string {
  const f = facts(g)
  const name = escapeHTML(g.name)
  const appId = steamAppId(g)
  const sources = (g.sources ?? []).map((s) =>
    brandIcon(s.id, named([s.id], SOURCE_NAMES)[0] ?? s.id, sourceURL(s)),
  )
  const stores = Object.keys(g.stores ?? {}).map((id) =>
    brandIcon(
      id,
      named([id], STORE_NAMES)[0] ?? id,
      id === 'steam' && appId ? `https://store.steampowered.com/app/${appId}/` : undefined,
    ),
  )
  const install = f.loader
    ? `Pick ${name}. Mortar installs ${escapeHTML(f.loader)} and makes your first profile.`
    : `Pick ${name}. Mortar makes your first profile and keeps its mods apart from every other profile's.`
  const more = neighbours(g, games)
  const hero = banner(
    g,
    art,
    `<h1>${name}</h1>
        <p class="lede">Mod ${name} with Mortar, the free and open source mod manager for Linux, Windows and Steam Deck.</p>
        <div class="cta ext-cta">
          <a class="btn" href="/download/">Download Mortar</a>
          <a class="btn-ghost" href="/guide/">User guide</a>
        </div>`,
  )
  const loader = f.loader
    ? `<span class="fact-icon">${escapeHTML(f.loader)}</span>`
    : '<span class="fact-icon">None needed</span>'
  const body = `      <div class="facts-strip">
        <div class="facts-group"><span class="facts-label">Mod loader</span>${loader}</div>
        <span class="facts-sep" aria-hidden="true"></span>
        <div class="facts-group"><span class="facts-label">Mods from</span>${sources.join('')}</div>
        <span class="facts-sep" aria-hidden="true"></span>
        <div class="facts-group"><span class="facts-label">Finds the game in</span>${stores.join('')}</div>
      </div>
      <h2>Start modding ${name}</h2>
      <ol class="steps">
        <li><a href="/download/">Download Mortar</a> for Linux or Windows and open it.</li>
        <li>${install}</li>
        <li>Open Browse, add the mods you want, and press Play. Each profile keeps its own mods and settings.</li>
      </ol>
      <h2>Share a profile</h2>
      <p>A profile for ${name} shared from Mortar is a link to <code>mortar.rethunk.tech/${g.id}/p</code>. Whoever opens it sees the mod list and imports it into Mortar in one click. The whole profile is inside the link, so nothing is uploaded anywhere.</p>${
        more.games.length > 0
          ? `
      <div data-pagefind-ignore>
      <h2>${more.title}</h2>
      <ul class="game-grid">
${more.games.map((o) => gameCard(o, art)).join('\n')}
      </ul>
      <p><a href="/games/">All ${games.length} supported games</a></p>
      </div>`
          : ''
      }`
  const loaderText = f.loader ? `installs ${f.loader}, ` : ''
  return page({
    path: `site/${g.id}/index.html`,
    title: `${g.name} mod manager: Mortar`,
    description: `Mortar is a free, open source mod manager for ${g.name} on Linux, Windows and Steam Deck: it ${loaderText}keeps each set of mods in its own profile and downloads from ${f.sources}.`,
    body,
    hero,
    jsonLd: {
      '@context': 'https://schema.org',
      '@type': 'BreadcrumbList',
      itemListElement: [
        { '@type': 'ListItem', position: 1, name: 'Mortar', item: `${SITE}/` },
        { '@type': 'ListItem', position: 2, name: 'Games', item: `${SITE}/games/` },
        { '@type': 'ListItem', position: 3, name: g.name, item: `${SITE}/${g.id}/` },
      ],
    },
  })
}

/** The page a shared profile link opens (`/<game>/p#<payload>`): site/share/app.js reads the payload in the browser. */
export function sharePage(g: Game, art: Record<string, SteamArt>): string {
  const name = escapeHTML(g.name)
  return page({
    path: `site/${g.id}/p/index.html`,
    title: `Mortar: shared ${g.name} profile`,
    description: `A ${g.name} mod profile shared from Mortar.`,
    index: false,
    css: '/share/share.css',
    module: '/share/app.js',
    hero: banner(
      g,
      art,
      `<h1>Shared ${name} profile</h1>`,
    ),
    body: `      <p id="error" class="card" hidden></p>
      <noscript><p class="card">This page needs JavaScript to read the shared profile.</p></noscript>
      <section id="profile" hidden>
        <p id="name" class="title"></p>
        <p class="muted"><span id="count"></span>. Mortar resolves mod names when you import.</p>
        <label id="versionRow" hidden>Your game version
          <input id="version" inputmode="decimal" autocomplete="off" />
          <span id="flagged" class="muted"></span>
        </label>
        <ul id="list" data-marks="${escapeHTML(JSON.stringify(shareMarks()))}"></ul>
      </section>
      <div class="actions">
        <a id="open" class="btn" href="mortar://" hidden>Open in Mortar</a>
        <a id="download" class="btn-ghost" href="/download/" hidden>Get Mortar</a>
      </div>
      <p id="copied" class="muted" hidden>Link copied. After installing Mortar, paste it into Import.</p>
      <h2>What is Mortar?</h2>
      <p>A free, open source mod manager for Linux, Windows and Steam Deck. It installs each game's mod loader, keeps each set of mods in its own profile and downloads every mod from its own page. <a href="/${g.id}/">How Mortar handles ${name}</a></p>
      <p class="muted">Everything on this page happens in your browser; the profile is never sent anywhere. Pictures of GitHub and Thunderstore mods load from those sites.</p>`,
  })
}

function options(label: string, id: string, values: string[]): string {
  const opts = [...new Set(values)].sort((a, b) => a.localeCompare(b))
  return `<label>${label}
          <select id="${id}"><option value="">Any</option>${opts.map((v) => `<option>${escapeHTML(v)}</option>`).join('')}</select>
        </label>`
}

/** /games/: every playable game as a card, filtered in the browser by name, loader, mod site and store. */
export function gamesIndexPage(games: Game[], art: Record<string, SteamArt>): string {
  const loaderOf = (g: Game) => facts(g).loader || 'No loader needed'
  const sourcesOf = (g: Game) =>
    named(
      (g.sources ?? []).map((s) => s.id),
      SOURCE_NAMES,
    )
  const storesOf = (g: Game) => named(Object.keys(g.stores ?? {}), STORE_NAMES)
  const sorted = [...games].sort((a, b) => a.name.localeCompare(b.name))
  const cards = sorted
    .map((g) =>
      gameCard(
        g,
        art,
        ` data-name="${escapeHTML(g.name.toLowerCase())}" data-loader="${escapeHTML(loaderOf(g))}" data-sources="${escapeHTML(sourcesOf(g).join('|'))}" data-stores="${escapeHTML(storesOf(g).join('|'))}"`,
      ),
    )
    .join('\n')
  return page({
    path: 'site/games/index.html',
    title: 'Games Mortar supports',
    description: `The ${games.length} games Mortar manages mods for, with each game's mod loader, mod sites and stores.`,
    script: '/games.js',
    body: `      <h1 class="ext-title">Supported games</h1>
      <p class="lede">${games.length} games, each with its mod loader and mod sites set up for you. Pick one to see how Mortar handles it.</p>
      <div class="game-filters">
        <label>Search <input type="search" id="game-q" autocomplete="off" placeholder="Game name" /></label>
        ${options('Mod loader', 'game-loader', games.map(loaderOf))}
        ${options('Mod site', 'game-source', games.flatMap(sourcesOf))}
        ${options('Store', 'game-store', games.flatMap(storesOf))}
      </div>
      <p class="game-count" id="game-count" aria-live="polite"></p>
      <ul class="game-grid" id="game-grid">
${cards}
      </ul>`,
  })
}

const BAND = 12

/** The landing page's Games section: the first games in catalog order, the best known, and a link to the rest. */
export function gamesBand(games: Game[], art: Record<string, SteamArt>): string {
  return `
      <section class="block" id="games">
        <div class="center">
          <p class="kicker">Games</p>
          <h2>${games.length} games, ready to mod</h2>
          <p class="lede">Each one with its mod loader and mod sites set up for you.</p>
        </div>
        <ul class="game-grid band">
${games
  .slice(0, BAND)
  .map((g) => gameCard(g, art))
  .join('\n')}
        </ul>
        <p class="center band-more"><a class="btn-ghost" href="/games/">All ${games.length} games</a></p>
      </section>
      `
}
