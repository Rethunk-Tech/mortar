import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { basename, dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { CDPSession, Page } from '@playwright/test'
import {
  hookScript,
  Meter,
  muiPreset,
  nestedBoxes,
  unevenSpacing,
} from '@rethunk/react-render-audit'

const TOP = Number(process.env.MORTAR_RENDER_TOP ?? '8')
const ALL_NAMES = process.env.MORTAR_RENDER_ALL_NAMES === '1'
const BINDINGS = fileURLToPath(new URL('../bindings', import.meta.url))
const BINDING_CALL = /export function (\w+)\(.*\{\n\s*return \$Call\.ByID\((\d+)/g

// A backend call names its method by a numeric id; the generated bindings say which service method each id is.
function methodNames(): Map<number, string> {
  const out = new Map<number, string>()
  for (const file of readdirSync(BINDINGS, { recursive: true, encoding: 'utf8' })) {
    if (!file.endsWith('service.ts')) {
      continue
    }
    const service = basename(dirname(file))
    for (const [, name, id] of readFileSync(join(BINDINGS, file), 'utf8').matchAll(BINDING_CALL)) {
      out.set(Number(id), `${service}.${name}`)
    }
  }
  return out
}

const METHODS = methodNames()

export function methodOf(body: string): string {
  try {
    const id = (JSON.parse(body) as { args?: { methodID?: number } }).args?.methodID
    return id === undefined ? 'unknown' : (METHODS.get(id) ?? String(id))
  } catch {
    return 'unknown'
  }
}

export const HOOK = hookScript({ ignoreCommitsOf: muiPreset.ignoreCommitsOf })
export const mortarBoxes = (page: Page) => nestedBoxes(page, muiPreset)
export const mortarGaps = (page: Page) => unevenSpacing(page, muiPreset)

export function mortarMeter(page: Page, cdp: CDPSession): Meter {
  return new Meter(page, cdp, {
    backendCall: (r) =>
      new URL(r.url()).pathname.startsWith('/wails/') ? methodOf(r.postData() ?? '') : null,
    top: TOP,
    libraryNames: ALL_NAMES ? null : muiPreset.libraryNames,
    log: console.log,
  })
}

/** Dismisses the toasts on screen, whose timers would otherwise commit during the case, and logs what they said. */
export async function quietToasts(page: Page, name: string) {
  const region = page.getByRole('region', { name: 'Notifications' })
  const dismiss = region.getByRole('button', { name: 'Dismiss' })
  if ((await dismiss.count()) === 0) {
    return
  }
  console.log(`${name}: cleared toasts: ${(await region.innerText()).replace(/\s+/g, ' ').trim()}`)
  for (let left = await dismiss.count(); left > 0; left--) {
    await dismiss.first().click()
  }
}

/** A piece of text on a render's picture, in CSS pixels of the screenshot. A label is an aria-label, title, placeholder or
 * alt that the picture does not show, such as an icon button's tooltip; its box is the element that carries it. */
export interface ShotText {
  text: string
  kind: 'text' | 'label'
  x: number
  y: number
  w: number
  h: number
}

const LABELS = ['aria-label', 'title', 'placeholder', 'alt']

/** Every visible element's own text and labels on the page as the screenshot shows it, so a string can be traced to the
 * render and the spot that shows it without reading the picture. */
export const shotText = (page: Page): Promise<ShotText[]> =>
  page.evaluate((labels) => {
    const out: ShotText[] = []
    const flat = (s: string) => s.replace(/\s+/gu, ' ').trim()
    for (const el of document.body.querySelectorAll<HTMLElement>('*')) {
      if (!el.checkVisibility({ opacityProperty: true, visibilityProperty: true })) {
        continue
      }
      const r = el.getBoundingClientRect()
      if (
        r.width === 0 ||
        r.height === 0 ||
        r.bottom <= 0 ||
        r.right <= 0 ||
        r.top >= innerHeight ||
        r.left >= innerWidth
      ) {
        continue
      }
      const box = {
        x: Math.round(r.x),
        y: Math.round(r.y),
        w: Math.round(r.width),
        h: Math.round(r.height),
      }
      const own = [...el.childNodes].some(
        (n) => n.nodeType === Node.TEXT_NODE && flat(n.textContent ?? '') !== '',
      )
      if (own) {
        out.push({ text: flat(el.textContent ?? ''), kind: 'text', ...box })
      }
      for (const name of labels) {
        const value = flat(el.getAttribute(name) ?? '')
        if (value !== '') {
          out.push({ text: value, kind: 'label', ...box })
        }
      }
    }
    return out
  }, LABELS)

/** The page's picture at `<base>.png` and the text it shows at `<base>.text.json`. */
export async function saveShot(page: Page, base: string): Promise<void> {
  await page.screenshot({ path: `${base}.png` })
  writeFileSync(`${base}.text.json`, `${JSON.stringify(await shotText(page))}\n`)
}
