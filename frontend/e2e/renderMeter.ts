import { readdirSync, readFileSync } from 'node:fs'
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
