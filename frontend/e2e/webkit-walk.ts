// The render budget's layout check and screenshots in WebKitGTK, the engine Mortar draws with on Linux. Chromium (the
// budget's own walk, and WebView2 on Windows) lays some CSS out differently: a stretched flex item with an aspect ratio
// collapsed to nothing in WebKitGTK only. Drives MiniBrowser through WebKitWebDriver (both ship with webkit2gtk) on the
// sandbox's hidden display, one session and one load, reaching each screen from the one before. `bun run renders` runs
// it after its own walk on the same sandbox; alone, against a running sandbox:
//   MORTAR_SELFTEST_DIR=/var/tmp/mortar-selftest MORTAR_RENDER_SHOTS=/var/tmp/shots bun run renders:webkit
// MORTAR_RENDER_URL names the server and MORTAR_RENDER_ONLY=a,b the screens whose names start with these. Exits 1 on a
// layout break. React render counts need Chromium's DevTools protocol, so this pass only checks and pictures.
import { existsSync, mkdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import process from 'node:process'
import { collapsed, covered, overflow } from './renderLayout.ts'

const PORT = 4471
const DRIVER = `http://127.0.0.1:${PORT}`
const SETTLE_MS = 3000
const VIEW = { width: 1400, height: 840 }
const DRIVER_BIN = '/usr/bin/WebKitWebDriver'

export interface Walk {
  sandbox: string
  url: string
  shots?: string | undefined
  only: string[]
}

// A step clicks the element of that role whose accessible name matches, in the page.
interface Step {
  role: 'tab' | 'button' | 'menuitem'
  name: string
  // Searches only inside this container, for a name another control shares (the bell is also "Notifications").
  within?: string
}
// The screens in an order where each is reached from the one before by clicking, so the app loads once. from is what
// leads back to a screen's starting point after a reload, when a step finds nothing to click.
interface Screen {
  name: string
  steps: Step[]
  from?: Step[]
}
const tab = (name: string): Step[] => [{ role: 'tab', name: `^${name}` }]
const OPEN_SETTINGS: Step[] = [
  { role: 'button', name: '^Mortar menu$' },
  { role: 'menuitem', name: '^Mortar settings' },
]
const SECTIONS = [
  'General',
  'Appearance',
  'Mods and profiles',
  'Downloads',
  'Accounts',
  'Source health',
  'Updates',
  'Notifications',
  'Storage',
  'Launchers',
  'Shortcuts',
  'About',
]
const view = (which: string): Step => ({ role: 'button', name: `^${which} view$` })
const SCREENS: Screen[] = [
  { name: 'home', steps: tab('Home') },
  { name: 'mods-grid', steps: [...tab('Mods'), view('Grid')] },
  { name: 'mods-list', steps: [...tab('Mods'), view('List')] },
  ...['Saves', 'Config', 'Screenshots', 'Problems', 'Load order', 'Performance', 'Console'].map(
    (t) => ({
      name: t.toLowerCase().replace(' ', '-'),
      steps: tab(t),
    }),
  ),
  { name: 'browse-grid', steps: [...tab('Browse'), view('Grid')] },
  { name: 'browse-list', steps: [...tab('Browse'), view('List')] },
  ...SECTIONS.map((s, i) => ({
    name: `settings-${s.toLowerCase().replaceAll(' ', '-')}`,
    steps: [
      ...(i === 0 ? OPEN_SETTINGS : []),
      { role: 'button' as const, name: `^${s}$`, within: '[aria-label="Settings sections"]' },
    ],
    from: OPEN_SETTINGS,
  })),
  {
    name: 'game-select',
    steps: [
      { role: 'button', name: '^(Switch game|Choose a game)' },
      { role: 'menuitem', name: '^All games' },
    ],
  },
]

type Json = Record<string, unknown>
const isJson = (v: unknown): v is Json => typeof v === 'object' && v !== null

async function call(method: string, path: string, body?: Json): Promise<unknown> {
  const res = await fetch(DRIVER + path, {
    method,
    headers: { 'content-type': 'application/json' },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })
  const json: unknown = await res.json()
  if (!(res.ok && isJson(json))) {
    throw new Error(`${method} ${path}: ${JSON.stringify(json).slice(0, 300)}`)
  }
  return json.value
}

// The page-side click: the first element of the role whose aria-label or text matches, waiting up to a few seconds.
const CLICK = `
  const [role, pattern, within, done] = arguments
  const re = new RegExp(pattern)
  const sel = { tab: '[role=tab]', button: 'button,[role=button]', menuitem: '[role=menuitem]' }[role]
  const start = Date.now()
  const tryClick = () => {
    const el = [...(within ? document.querySelector(within) ?? document : document).querySelectorAll(sel)].find((e) => re.test((e.getAttribute('aria-label') ?? e.textContent ?? '').trim()))
    if (el) { el.click(); done(true) } else if (Date.now() - start > 8000) { done(false) } else { setTimeout(tryClick, 200) }
  }
  tryClick()
`
const check = (fn: () => unknown) => `return (${fn.toString()})()`

// MiniBrowser's environment: the sandbox's hidden X display only (its Xwayland), with Wayland and every other display
// removed, so a missing sandbox fails instead of opening on the user's screen.
function sandboxEnv(sandbox: string): Record<string, string> | null {
  const envFile = join(sandbox, 'run', 'display.env')
  if (!existsSync(envFile)) {
    return null
  }
  const env: Record<string, string> = {}
  for (const [k, v] of Object.entries(process.env)) {
    if (v !== undefined && k !== 'DISPLAY' && k !== 'WAYLAND_DISPLAY') {
      env[k] = v
    }
  }
  for (const line of readFileSync(envFile, 'utf8').split('\n')) {
    const [k, ...v] = line.split('=')
    if (k && k !== 'WAYLAND_DISPLAY') {
      env[k] = v.join('=')
    }
  }
  return { ...env, GDK_BACKEND: 'x11' }
}

const click = async (id: string, s: Step) =>
  Boolean(
    await call('POST', `/session/${id}/execute/async`, {
      script: CLICK,
      args: [s.role, s.name, s.within ?? ''],
    }),
  )

async function load(id: string, url: string) {
  await call('POST', `/session/${id}/url`, { url })
  await Bun.sleep(SETTLE_MS)
}

// Reaches one screen from the one before, reloading only when a step finds nothing, and returns its layout breaks.
async function visit(id: string, screen: Screen, w: Walk): Promise<string[]> {
  const breaks: string[] = []
  let reached = true
  for (const s of screen.steps) {
    reached = reached && (await click(id, s))
    await Bun.sleep(SETTLE_MS / 2)
  }
  if (!reached) {
    await load(id, w.url)
    for (const s of [
      ...(screen.from ?? []),
      ...screen.steps.filter((x) => !(screen.from ?? []).includes(x)),
    ]) {
      if (!(await click(id, s))) {
        breaks.push(`LAYOUT ${screen.name} | missing | ${s.role} ${s.name} not found`)
      }
      await Bun.sleep(SETTLE_MS / 2)
    }
  }
  await Bun.sleep(SETTLE_MS)
  for (const fn of [collapsed, covered, overflow]) {
    const found = await call('POST', `/session/${id}/execute/sync`, { script: check(fn), args: [] })
    for (const b of Array.isArray(found) ? found.filter(isJson) : []) {
      breaks.push(`LAYOUT ${screen.name} | ${b.kind} | ${b.where} | ${b.text}`)
    }
  }
  if (w.shots) {
    const png = await call('GET', `/session/${id}/screenshot`)
    await Bun.write(join(w.shots, `${screen.name}.png`), Buffer.from(String(png), 'base64'))
  }
  return breaks
}

// Walks every screen and returns its layout breaks, or null when WebKitGTK's driver or the sandbox's display is missing.
export async function webkitWalk(w: Walk): Promise<string[] | null> {
  const env = sandboxEnv(w.sandbox)
  if (!(env && existsSync(DRIVER_BIN))) {
    return null
  }
  if (w.shots) {
    mkdirSync(w.shots, { recursive: true })
  }
  const driver = Bun.spawn([DRIVER_BIN, `--port=${PORT}`], {
    env,
    stdout: 'ignore',
    stderr: 'ignore',
  })
  const breaks: string[] = []
  try {
    await Bun.sleep(800)
    const session = await call('POST', '/session', {
      capabilities: {
        alwaysMatch: {
          'webkitgtk:browserOptions': {
            binary: '/usr/libexec/webkit2gtk-4.1/MiniBrowser',
            args: ['--automation'],
          },
        },
      },
    })
    const id = isJson(session) ? String(session.sessionId) : ''
    await call('POST', `/session/${id}/window/rect`, VIEW).catch(() => undefined)
    await load(id, w.url)
    for (const screen of SCREENS.filter(
      (s) => w.only.length === 0 || w.only.some((o) => s.name.startsWith(o)),
    )) {
      breaks.push(...(await visit(id, screen, w)))
      console.log(`${screen.name} checked (WebKitGTK)`)
    }
    await call('DELETE', `/session/${id}`)
  } finally {
    driver.kill()
  }
  return breaks
}

if (import.meta.main) {
  const sandbox = process.env.MORTAR_SELFTEST_DIR ?? '/var/tmp/mortar-selftest'
  webkitWalk({
    sandbox,
    url: process.env.MORTAR_RENDER_URL ?? 'http://127.0.0.1:9455',
    shots: process.env.MORTAR_RENDER_SHOTS,
    only: (process.env.MORTAR_RENDER_ONLY ?? '').split(',').filter(Boolean),
  }).then(
    (breaks) => {
      if (breaks === null) {
        console.error(
          `no WebKitWebDriver, or no hidden display under ${sandbox}: start the sandbox first`,
        )
        process.exit(2)
      }
      console.log(breaks.length === 0 ? 'layout: no breaks (WebKitGTK)' : breaks.join('\n'))
      process.exit(breaks.length > 0 ? 1 : 0)
    },
    (e) => {
      console.error(e)
      process.exit(1)
    },
  )
}
