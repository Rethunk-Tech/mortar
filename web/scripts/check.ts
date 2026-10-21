// Checks the built site: every link and picture a page names on this site exists in the build, and every picture in
// public/img is used by some page, stylesheet or script, so none ships unused. The gate runs it after the build.
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const OUT = '../site'
const MAX_LISTED = 40
// Served beside the site rather than built by it: the package repositories and the search index (built at publish).
const ELSEWHERE = /^\/(packages|pagefind)\//

const files = readdirSync(OUT, { recursive: true, encoding: 'utf8' })
const pages = files.filter((f) => f.endsWith('.html'))
const texts = files
  .filter((f) => /\.(html|css|js|xml)$/.test(f))
  .map((f) => readFileSync(join(OUT, f), 'utf8'))

function exists(path: string): boolean {
  const clean = decodeURIComponent(path.replace(/[?#].*$/, ''))
  const target = join(OUT, clean)
  return clean.endsWith('/')
    ? existsSync(join(target, 'index.html'))
    : existsSync(target) || existsSync(join(target, 'index.html'))
}

const broken: string[] = []
for (const page of pages) {
  const html = readFileSync(join(OUT, page), 'utf8')
  for (const [, url = ''] of html.matchAll(/(?:href|src)="([^"]+)"/g)) {
    if (url.startsWith('/') && !url.startsWith('//') && !ELSEWHERE.test(url) && !exists(url)) {
      broken.push(`${page}: ${url}`)
    }
  }
}

const all = texts.join('\n')
const pictures = readdirSync('public/img', { recursive: true, encoding: 'utf8' }).filter((f) =>
  /\.(webp|png|jpe?g|svg|avif|gif)$/.test(f),
)
const unused = pictures.filter((f) => !all.includes(`/img/${f}`))

console.log(
  `site check: ${pages.length} pages, ${broken.length} broken links, ${unused.length} unused pictures`,
)
for (const b of broken.slice(0, MAX_LISTED)) {
  console.error(`broken: ${b}`)
}
for (const u of unused) {
  console.error(`unused picture: public/img/${u}`)
}
if (broken.length > 0 || unused.length > 0) {
  process.exit(1)
}
