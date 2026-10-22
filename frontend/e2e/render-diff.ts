// Compares two MORTAR_RENDER_SHOTS folders screen by screen, for a person to look at what changed between two runs (two
// commits, or the last release and now):
//   bun run renders:diff /var/tmp/shots-v0.3.11 /var/tmp/shots-head
// Lists each screen whose pixels differ, most changed first, and writes <new>/diff/<case>.png with the changed pixels in
// red over a faded copy of the new screen. A screen only one side has is listed too. Exits 0; it reports, it does not
// judge, since copied data that changed between runs (a new mod, a date) also moves pixels.
import { existsSync, mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import process from 'node:process'
import { chromium } from '@playwright/test'

// A channel that moves less than this is antialiasing or a colour rounding, not a change.
const CHANNEL_TOLERANCE = 24
const PERCENT = 100

const [oldDir, newDir] = process.argv.slice(2)
if (!(oldDir && newDir && existsSync(oldDir) && existsSync(newDir))) {
  console.error('usage: bun run renders:diff OLD_SHOTS_DIR NEW_SHOTS_DIR')
  process.exit(2)
}
const pngs = (dir: string) => new Set(readdirSync(dir).filter((f) => f.endsWith('.png')))
const before = pngs(oldDir)
const after = pngs(newDir)
const both = [...after].filter((f) => before.has(f))
const out = join(newDir, 'diff')
mkdirSync(out, { recursive: true })

const browser = await chromium.launch()
const page = await browser.newPage()
const rows: { name: string; changed: number }[] = []
for (const file of both) {
  const dataUrl = (dir: string) =>
    `data:image/png;base64,${readFileSync(join(dir, file)).toString('base64')}`
  const result = await page.evaluate(
    async ({ a, b, tolerance }) => {
      const load = async (src: string) => {
        const img = new Image()
        img.src = src
        await img.decode()
        return img
      }
      const [x, y] = await Promise.all([load(a), load(b)])
      const w = Math.max(x.width, y.width)
      const h = Math.max(x.height, y.height)
      const pixels = (img: HTMLImageElement) => {
        const c = new OffscreenCanvas(w, h)
        const ctx = c.getContext('2d') as OffscreenCanvasRenderingContext2D
        ctx.drawImage(img, 0, 0)
        return ctx.getImageData(0, 0, w, h)
      }
      const pa = pixels(x).data
      const shown = pixels(y)
      const pb = shown.data
      let changed = 0
      for (let i = 0; i < pa.length; i += 4) {
        const moved =
          Math.abs((pa[i] ?? 0) - (pb[i] ?? 0)) > tolerance ||
          Math.abs((pa[i + 1] ?? 0) - (pb[i + 1] ?? 0)) > tolerance ||
          Math.abs((pa[i + 2] ?? 0) - (pb[i + 2] ?? 0)) > tolerance
        if (moved) {
          changed += 1
          pb.set([255, 0, 0, 255], i)
        } else {
          pb[i + 3] = 70
        }
      }
      const c = new OffscreenCanvas(w, h)
      const ctx = c.getContext('2d') as OffscreenCanvasRenderingContext2D
      ctx.putImageData(shown, 0, 0)
      const blob = await c.convertToBlob({ type: 'image/png' })
      const bytes = new Uint8Array(await blob.arrayBuffer())
      let binary = ''
      for (const byte of bytes) {
        binary += String.fromCharCode(byte)
      }
      return { changed: changed / (w * h), png: btoa(binary) }
    },
    { a: dataUrl(oldDir), b: dataUrl(newDir), tolerance: CHANNEL_TOLERANCE },
  )
  if (result.changed > 0) {
    writeFileSync(join(out, file), Buffer.from(result.png, 'base64'))
    rows.push({ name: file.replace(/\.png$/, ''), changed: result.changed })
  }
}
await browser.close()

rows.sort((a, b) => b.changed - a.changed)
for (const r of rows) {
  console.log(`${(r.changed * PERCENT).toFixed(2).padStart(6)}%  ${r.name}`)
}
for (const f of [...before].filter((name) => !after.has(name))) {
  console.log(` gone    ${f.replace(/\.png$/, '')}`)
}
for (const f of [...after].filter((name) => !before.has(name))) {
  console.log(`  new    ${f.replace(/\.png$/, '')}`)
}
console.log(`${rows.length} of ${both.length} screens changed; pictures of the changes in ${out}`)
