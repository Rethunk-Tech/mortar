// After the build, draws each page's link-preview picture from its own title and description into /og/<path>.png,
// the address the page's og:image names (src/lib/seo.ts). Pictures are cached by their text, so a rebuild redraws only
// the pages whose title or description changed.
import { createHash } from 'node:crypto'
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  writeFileSync,
} from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { Resvg } from '@resvg/resvg-js'
import type { AstroIntegration } from 'astro'
import satori from 'satori'

const WIDTH = 1200
const HEIGHT = 630
const LONG_TITLE_CHARS = 48
const LONG_TITLE_PX = 56
const TITLE_PX = 68
const DESCRIPTION_MAX = 160
const DESCRIPTION_CUT = 157
const CACHE = 'node_modules/.cache/og'
// The running-bond tile of Mortar's banners, darkened: rounded bricks a shade above the card's own colour.
const BRICK_TILE = `data:image/svg+xml;base64,${Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" width="56" height="60"><rect x="4" y="4" width="48" height="22" rx="4" fill="#1F1F26"/><rect x="-24" y="34" width="48" height="22" rx="4" fill="#1F1F26"/><rect x="32" y="34" width="48" height="22" rx="4" fill="#1F1F26"/></svg>').toString('base64')}`
// A picture is redrawn when its words or this script change.
const DRAWER_HASH_CHARS = 12
const DRAWER = createHash('sha256')
  .update(readFileSync(fileURLToPath(import.meta.url)))
  .digest('hex')
  .slice(0, DRAWER_HASH_CHARS)
const FONTS =
  'node_modules/@fontsource/atkinson-hyperlegible-next/files/atkinson-hyperlegible-next-latin'

const decodeEntities = (s: string) =>
  s
    .replace(/&#(\d+);/g, (_, n: string) => String.fromCharCode(Number(n)))
    .replaceAll('&quot;', '"')
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&amp;', '&')

const meta = (html: string, property: string) =>
  new RegExp(`<meta property="${property}" content="([^"]*)"`).exec(html)?.[1]

interface Node {
  type: string
  props: {
    style?: Record<string, string | number>
    children?: (Node | string)[] | Node | string
    src?: string
  }
}

const el = (
  type: string,
  style: Record<string, string | number>,
  children?: Node['props']['children'],
): Node => ({
  type,
  props: { style, ...(children === undefined ? {} : { children }) },
})

function card(title: string, description: string, icon: string): Node {
  const big = title.length > LONG_TITLE_CHARS ? LONG_TITLE_PX : TITLE_PX
  return el(
    'div',
    {
      width: WIDTH,
      height: HEIGHT,
      boxSizing: 'border-box',
      display: 'flex',
      flexDirection: 'column',
      justifyContent: 'space-between',
      padding: '64px 72px',
      backgroundColor: '#19191E',
      backgroundImage: `url(${BRICK_TILE})`,
      backgroundRepeat: 'repeat',
      backgroundSize: '56px 60px',
      color: 'rgba(255,255,255,0.93)',
      fontFamily: 'Atkinson',
      borderBottom: '10px solid #d6b17a',
    },
    [
      el('div', { display: 'flex', alignItems: 'center', gap: 18, fontSize: 36, fontWeight: 800 }, [
        { type: 'img', props: { src: icon, style: { width: 64, height: 64 } } },
        'Mortar',
      ]),
      el('div', { display: 'flex', flexDirection: 'column', gap: 22 }, [
        el(
          'div',
          { fontSize: big, fontWeight: 800, lineHeight: 1.05, letterSpacing: -1, maxWidth: 1040 },
          title,
        ),
        el(
          'div',
          { fontSize: 30, color: 'rgba(210,210,215,0.82)', lineHeight: 1.35, maxWidth: 1000 },
          description,
        ),
      ]),
      el(
        'div',
        { display: 'flex', fontSize: 24, color: '#d6b17a', fontWeight: 800 },
        'mortar.rethunk.tech',
      ),
    ],
  )
}

/** The page's own words for the picture: its title without the site name, and a description cut to fit. */
function words(html: string): { title: string; description: string } | undefined {
  const title = meta(html, 'og:title')
  const image = meta(html, 'og:image')
  if (!(title && image)) {
    return undefined
  }
  const description = decodeEntities(meta(html, 'og:description') ?? '')
  // "Mortar: <what it is>" titles read badly cut; their page heading says it better.
  const h1 = /<h1[^>]*>(.*?)<\/h1>/s
    .exec(html)?.[1]
    ?.replace(/<br\s*\/?>/g, ' ')
    .replace(/<[^>]+>/g, '')
    // One pass can leave a tag's start behind (`<scr<x>ipt`); a heading's text has no `<` of its own.
    .replace(/</g, '')
    .replace(/\s+/g, ' ')
    .trim()
  const own = decodeEntities(title)
  return {
    title: own.startsWith('Mortar: ') && h1 ? decodeEntities(h1) : own.replace(/: Mortar$/, ''),
    description:
      description.length > DESCRIPTION_MAX
        ? `${description.slice(0, DESCRIPTION_CUT).replace(/\s+\S*$/, '')}…`
        : description,
  }
}

function pages(dir: string): string[] {
  return readdirSync(dir, { recursive: true, encoding: 'utf8' }).filter((f) => f.endsWith('.html'))
}

export function og(): AstroIntegration {
  return {
    name: 'mortar-og',
    hooks: {
      'astro:build:done': async ({ dir, logger }) => {
        const out = fileURLToPath(dir)
        const fonts = [
          {
            name: 'Atkinson',
            data: readFileSync(`${FONTS}-400-normal.woff`),
            weight: 400 as const,
          },
          {
            name: 'Atkinson',
            data: readFileSync(`${FONTS}-800-normal.woff`),
            weight: 800 as const,
          },
        ]
        const icon = `data:image/png;base64,${readFileSync('public/img/icon-192.png').toString('base64')}`
        mkdirSync(CACHE, { recursive: true })
        let drawn = 0
        let cached = 0
        for (const file of pages(out)) {
          const html = readFileSync(join(out, file), 'utf8')
          const w = words(html)
          const image = meta(html, 'og:image')
          if (!(w && image)) {
            continue
          }
          const target = join(out, new URL(image).pathname)
          const key = join(
            CACHE,
            `${createHash('sha256')
              .update(JSON.stringify([w, DRAWER]))
              .digest('hex')
              .slice(0, 24)}.png`,
          )
          if (existsSync(key)) {
            cached += 1
          } else {
            const svg = await satori(
              card(w.title, w.description, icon) as Parameters<typeof satori>[0],
              {
                width: WIDTH,
                height: HEIGHT,
                fonts,
              },
            )
            // Satori draws text as paths, so resvg needs no fonts; scanning the system's costs ~100 ms a picture.
            writeFileSync(
              key,
              new Resvg(svg, { font: { loadSystemFonts: false } }).render().asPng(),
            )
            drawn += 1
          }
          mkdirSync(dirname(target), { recursive: true })
          copyFileSync(key, target)
        }
        logger.info(`link-preview pictures: ${drawn} drawn, ${cached} from cache`)
      },
    },
  }
}
