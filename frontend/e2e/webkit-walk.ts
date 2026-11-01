// The render budget's layout check and screenshots in WebKitGTK, the engine Mortar draws with on Linux. Chromium (the
// budget's own walk, and WebView2 on Windows) lays some CSS out differently: a stretched flex item with an aspect ratio
// collapsed to nothing in WebKitGTK only. Drives MiniBrowser through WebKitWebDriver (both ship with webkit2gtk) on the
// sandbox's hidden display, one session and one load, reaching each screen from the one before. `bun run renders` runs
// it after its own walk on the same sandbox; alone, against a running sandbox:
//   MORTAR_SELFTEST_DIR=/var/tmp/mortar-selftest MORTAR_RENDER_SHOTS=/var/tmp/shots bun run renders:webkit
// MORTAR_RENDER_URL names the server and MORTAR_RENDER_ONLY=a,b the screens whose names start with these. Exits 1 on a
// layout break. React render counts need Chromium's DevTools protocol, so this pass only checks and pictures.
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import process from 'node:process'
import { type Screen, type Step, webkitWalk as walk } from '@rethunk/react-render-audit'

export interface Walk {
  sandbox: string
  url: string
  shots?: string | undefined
  only: string[]
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

// Walks every screen and returns its layout breaks, or null when WebKitGTK's driver or the sandbox's display is missing.
export async function webkitWalk(w: Walk): Promise<string[] | null> {
  const env = sandboxEnv(w.sandbox)
  if (!env) {
    return null
  }
  const breaks = await walk({
    url: w.url,
    screens: SCREENS.filter((s) => w.only.length === 0 || w.only.some((o) => s.name.startsWith(o))),
    env,
    ...(w.shots ? { shots: w.shots } : {}),
    log: console.log,
  })
  return breaks?.map((b) => `LAYOUT ${b.screen} | ${b.kind} | ${b.where} | ${b.text}`) ?? null
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
