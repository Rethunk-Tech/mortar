// The parts every page of the site shares: nav, footer, search and link-preview metadata, and the page frame the
// generated pages are written in.

export const SITE = 'https://mortar.rethunk.tech'
export const REPO = 'https://github.com/Rethunk-Tech/mortar'
export const DISCORD = 'https://discord.gg/TfecaTm8J2'
const OG_IMAGE = {
  url: `${SITE}/img/og-dbcbd71467.jpg`,
  alt: "Mortar's game select: the supported games as a grid of cover art",
}

// What differs between pages: where the site root is from the page, which nav link is the current page, and which
// extra links the footer carries.
export interface Page {
  path: string
  root: string
  current?: 'download' | 'extension' | 'guide' | 'games' | 'faq'
  footer: { agpl?: boolean; source?: string }
}

export function escapeHTML(raw: string): string {
  return raw.replace(/[&<>"']/g, (c) => `&#${c.charCodeAt(0)};`)
}

export function listed(names: string[]): string {
  if (names.length <= 1) {
    return names.join('')
  }
  return `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`
}

/** The public address of a file under site/: its folder for an index.html. */
export function urlOf(path: string): string {
  return `${SITE}/${path.replace(/^site\//, '').replace(/index\.html$/, '')}`
}

export function nav(p: Page): string {
  const home = p.root ? '/' : ''
  const links: [href: string, label: string, current?: string][] = [
    ['/games/', 'Games', 'games'],
    [`${home}#features`, 'Features'],
    ['/extension/', 'Extension', 'extension'],
    ['/guide/', 'Guide', 'guide'],
    ['/faq/', 'FAQ', 'faq'],
  ]
  const outbound: [href: string, label: string, mark: string][] = [
    [REPO, 'GitHub', 'github'],
    [DISCORD, 'Discord', 'discord'],
  ]
  const link = (href: string, label: string, cls: string, current?: string) =>
    `<a${cls ? ` class="${cls}"` : ''} href="${href}"${current && p.current === current ? ' aria-current="page"' : ''}>${label}</a>`
  const all = [...links, ...outbound.map(([h, l]) => [h, l] as [string, string])]
  return `<nav class="nav" aria-label="Main">
      <div class="wrap">
        <a class="brand" href="/">
          <svg width="28" height="28" aria-hidden="true"><use href="${p.root}img/icons.svg#brand" /></svg>
          Mortar
        </a>
        <div class="nav-links">
          ${links.map(([h, l, c]) => link(h, l, 'optional', c)).join('\n          ')}
          <span class="nav-sep optional" aria-hidden="true"></span>
          <span class="nav-tools">
          <button type="button" class="nav-search" aria-label="Search" aria-keyshortcuts="/ Control+K Meta+K">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" /></svg>
          </button>
          ${outbound.map(([h, l, id]) => `<a class="nav-icon optional" href="${h}" title="${l}"><svg width="18" height="18" aria-hidden="true"><use href="/img/brands.svg#${id}" /></svg><span class="visually-hidden">${l}</span></a>`).join('\n          ')}
          </span>
          <a class="nav-cta" href="/download/"${p.current === 'download' ? ' aria-current="page"' : ''}>Download</a>
          <button type="button" class="nav-menu" aria-label="Menu" aria-expanded="false" aria-controls="nav-drawer">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16" /></svg>
          </button>
        </div>
      </div>
      <div class="nav-drawer" id="nav-drawer" hidden>
        ${all.map(([h, l, c]) => link(h, l, '', c)).join('\n        ')}
      </div>
      <link rel="stylesheet" href="/nav.css" />
      <link rel="stylesheet" href="/search.css" />
      <script src="/search.js" defer></script>
      <script type="module" src="/nav.js"></script>
    </nav>`
}

export function footer(p: Page): string {
  const { agpl, source = REPO } = p.footer
  // Guide, Discord, Mortar's source and Download are in the nav on every page; the footer adds only what it lacks.
  const links = [
    '<a href="/changelog/">Changelog</a>',
    '<a href="/privacy/">Privacy</a>',
    '<a href="/code-signing/">Code signing</a>',
    ...(source === REPO ? [] : [`<a href="${source}">Source</a>`]),
    ...(agpl ? ['<a href="https://www.gnu.org/licenses/agpl-3.0.html">AGPL-3.0</a>'] : []),
  ]
  return `<footer class="foot">
      <div class="wrap">
        <span>Mortar by <a href="https://we.rethunk.tech/">Rethunk.Tech</a>. Not affiliated with any game developer or mod site.</span>
        <span>${links.join(' · ')}</span>
      </div>
    </footer>`
}

/**
 * The canonical address, link-preview and theme metadata of a page. A page that writes its own og:title or
 * og:description outside this block keeps it, and the block leaves that one out.
 */
export function seo(m: {
  url: string
  title: string
  description: string
  ownTitle?: boolean
  ownDescription?: boolean
  jsonLd?: object
  feed?: boolean
}): string {
  const meta = (property: string, content: string) =>
    `<meta property="${property}" content="${escapeHTML(content)}" />`
  return [
    `<link rel="canonical" href="${m.url}" />`,
    '<meta name="theme-color" content="#19191e" />',
    meta('og:type', 'website'),
    meta('og:site_name', 'Mortar'),
    meta('og:url', m.url),
    ...(m.ownTitle ? [] : [meta('og:title', m.title)]),
    ...(m.ownDescription ? [] : [meta('og:description', m.description)]),
    meta('og:image', OG_IMAGE.url),
    meta('og:image:type', 'image/jpeg'),
    meta('og:image:width', '1200'),
    meta('og:image:height', '630'),
    meta('og:image:alt', OG_IMAGE.alt),
    '<meta name="twitter:card" content="summary_large_image" />',
    ...(m.feed
      ? [
          `<link rel="alternate" type="application/atom+xml" title="Mortar releases" href="${SITE}/changelog/feed.xml" />`,
        ]
      : []),
    ...(m.jsonLd
      ? [`<script type="application/ld+json">${JSON.stringify(m.jsonLd)}</script>`]
      : []),
  ].join('\n    ')
}

/** A whole generated page, its links rooted at / so it reads the same from any address (the 404 page included). */
export function page(p: {
  path: string
  title: string
  description: string
  body: string
  current?: Page['current']
  index?: boolean
  jsonLd?: object
  feed?: boolean
  script?: string
  module?: string
  css?: string
  hero?: string
}): string {
  const frame: Page = {
    path: p.path,
    root: '/',
    footer: {},
    ...(p.current ? { current: p.current } : {}),
  }
  return `<!doctype html>
<!-- Written by scripts/site.ts; edit the generator, not this file. -->
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="referrer" content="no-referrer" />
    <meta name="color-scheme" content="dark" />${p.index === false ? '\n    <meta name="robots" content="noindex" />' : ''}
    <title>${escapeHTML(p.title)}</title>
    <meta name="description" content="${escapeHTML(p.description)}" />
    ${seo({ url: urlOf(p.path), title: p.title, description: p.description, ...(p.jsonLd ? { jsonLd: p.jsonLd } : {}), ...(p.feed ? { feed: true } : {}) })}
    <link rel="icon" href="/favicon.png" />
    <link rel="apple-touch-icon" href="/img/icon-192.png" />
    <link rel="stylesheet" href="/base.css" />
    <link rel="stylesheet" href="/landing.css" />
    <link rel="stylesheet" href="/download.css" />
    <link rel="stylesheet" href="/cards.css" />${p.css ? `\n    <link rel="stylesheet" href="${p.css}" />` : ''}
  </head>
  <body>
    ${nav(frame)}

${p.hero ?? ''}    <main class="wrap dl prose${p.hero ? ' after-bleed' : ''}"${p.index === false ? '' : ' data-pagefind-body'}>
${p.body}
    </main>

    ${footer(frame)}${p.script ? `\n    <script src="${p.script}" defer></script>` : ''}${p.module ? `\n    <script type="module" src="${p.module}"></script>` : ''}
  </body>
</html>
`
}
