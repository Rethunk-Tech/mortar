import process from 'node:process'
import type { Page, Route } from '@playwright/test'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import { patchCalls } from './renderCoverage.ts'
import type { dialogKit } from './renderDialogs.ts'
import { biggestProfileIds, openGame } from './renderProfile.ts'
import { stubCalls } from './renderStubbed.ts'

const NEXUS_ACCOUNT = 1_189_232_250
const START_SSO = 305_010_372
const TRASH_PROJECT = 400_259_103
const GAME_LIST = 2_813_804_068
const LAUNCH_STATUS = 239_586_243
const MEASURE_IN_GAME = 1_064_106_242
const PEEK_IN_GAME = 512_110_599

type Stub = (route: Route) => Promise<void>

/** Answers the named binding calls with the error Go sends for a refused call: its kind in the cause, and its detail. */
export function failCalls(failures: Record<number, { kind: string; detail?: unknown }>): Stub {
  return async (route) => {
    const body = route.request().postData() ?? ''
    for (const [id, cause] of Object.entries(failures)) {
      if (body.includes(`"methodID":${id}`)) {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ kind: 'RuntimeError', message: `[${cause.kind}] refused`, cause }),
        })
        return
      }
    }
    await route.fallback()
  }
}

interface LoaderEdit {
  id: string
  order?: boolean
  perf?: boolean
}
interface ListedGame {
  loaderId: string
  loaders: (LoaderEdit & Record<string, unknown>)[] | null
}

/** The game list with every game's loaders edited, which decides the tabs and tools a profile gets. */
const withLoaders = (edit: (l: LoaderEdit & Record<string, unknown>) => void) =>
  patchCalls<ListedGame[] | null>([GAME_LIST], (games) => {
    for (const g of games ?? []) {
      for (const l of g.loaders ?? []) {
        edit(l)
      }
    }
    return games
  })

const dataDir = () =>
  process.env.MORTAR_RENDER_DATA ?? `${process.env.HOME ?? ''}/.local/share/mortar`

/** Reloads the app under the stubs, which the game list read at startup then goes through. */
async function reloadUnder(p: Page, tabs: (p: Page) => ReturnType<Page['locator']>) {
  await p.reload()
  await tabs(p).waitFor({ timeout: 30_000 })
}

/** Stream overlay settings of a game whose loader has one. */
function overlayFieldsCase(
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case {
  return {
    // The Stardew overlay fields that came with the day-of-week, friend and pet readings.
    name: 'game-settings-overlay',
    setup: async (p) => {
      // An earlier case may have left Lethal Company open.
      await openGame(p, tabs(p), 'Stardew Valley')
      await tab(p, 'Home')
      await p
        .getByRole('button', { name: / settings$/ })
        .last()
        .click()
      await p.getByRole('button', { name: /^This profile: / }).waitFor()
    },
    run: async (p) => {
      await p.getByRole('button', { name: 'Streaming', exact: true }).click()
      await p.getByRole('button', { name: /^All .* profiles$/ }).click()
      const toggle = p.getByRole('switch', { name: 'Stream overlay' })
      if (!(await toggle.isChecked())) {
        await toggle.click()
      }
      for (const label of ['Weekday', 'Friend', 'Pet', 'Pet watered']) {
        await p.getByText(label, { exact: true }).first().waitFor()
      }
      await p.getByText('Pet watered', { exact: true }).first().scrollIntoViewIfNeeded()
    },
  }
}

/**
 * Four states the backend produces only on a busy account, a locked folder, a running game or another loader, stubbed
 * at the binding so the pictures show them: a sign-in already waiting for the browser, the trash toast naming the
 * programs holding a folder, the Stop measuring button of an in-game measurement, and a game page with no Load order.
 */
export function stateCases(
  { developTab, pick }: ReturnType<typeof dialogKit>,
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case[] {
  const busySignIn = failCalls({ [START_SSO]: { kind: 'busy' } })
  const signedOut = patchCalls<{ signedIn: boolean; name: string; premium: boolean }>(
    [NEXUS_ACCOUNT],
    (account) => ({ ...account, signedIn: false, name: '', premium: false }),
  )
  const locked = failCalls({
    [TRASH_PROJECT]: { kind: 'in_use', detail: { holders: ['Visual Studio Code', 'Rider'] } },
  })
  const noOrder = withLoaders((l) => {
    l.order = false
  })
  // A loader that is not SMAPI measures in game through its companion, which can be stopped.
  const bepinex = patchCalls<ListedGame[] | null>([GAME_LIST], (games) => {
    for (const g of games ?? []) {
      g.loaderId = 'bepinex'
      for (const l of g.loaders ?? []) {
        l.id = 'bepinex'
        l.perf = true
      }
    }
    return games
  })
  const ids = biggestProfileIds(dataDir())
  const running = patchCalls<{ state: string; profile: string }>([LAUNCH_STATUS], (status) => ({
    ...status,
    state: 'running',
    profile: ids?.profile ?? status.profile,
    since: Date.now(),
  }))
  const measuring = stubCalls({
    [MEASURE_IN_GAME]: { measured: true, stoppable: true },
    [PEEK_IN_GAME]: { measured: true, stoppable: true, rows: [], frame: null },
  })
  const under =
    (...stubs: Stub[]) =>
    async (p: Page, on: boolean) => {
      for (const s of stubs) {
        await (on ? p.route('**/wails/runtime', s) : p.unroute('**/wails/runtime', s))
      }
    }
  const loadOrderStubs = under(noOrder)
  const perfStubs = under(bepinex, running, measuring)
  return [
    {
      name: 'settings-accounts-nexus-sign-in-busy',
      setup: async (p) => {
        await p.route('**/wails/runtime', busySignIn)
        await p.route('**/wails/runtime', signedOut)
        await reloadUnder(p, tabs)
        await openSettings(p)
        await p
          .getByRole('navigation', { name: 'Settings sections' })
          .getByRole('button', { name: 'Accounts', exact: true })
          .click()
      },
      run: async (p) => {
        await p.getByRole('button', { name: /^Sign in to Nexus/ }).click()
        await p.getByText('Approve Mortar in your browser…').waitFor()
        await p.getByRole('button', { name: 'Cancel', exact: true }).waitFor()
      },
      teardown: async (p) => {
        await p.unroute('**/wails/runtime', busySignIn)
        await p.unroute('**/wails/runtime', signedOut)
        await reloadUnder(p, tabs)
      },
    },
    {
      name: 'develop-trash-in-use',
      setup: async (p) => {
        await p.route('**/wails/runtime', locked)
        await developTab(p)
      },
      run: async (p) => {
        await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
        await pick(p, /^Remove from list/)
        const d = p.getByRole('dialog').last()
        await d.getByRole('checkbox', { name: /^Also move its folder to the trash/ }).check()
        await d.getByRole('button', { name: 'Remove and move to trash' }).click()
        const toast = p.getByRole('region', { name: 'Notifications' })
        await toast.getByText('Could not move the folder to the trash').waitFor()
        await toast.getByRole('button', { name: 'Details' }).click()
        await toast.getByText(/In use by .*Visual Studio Code.*Rider/).waitFor()
      },
      teardown: (p) => p.unroute('**/wails/runtime', locked),
    },
    {
      name: 'console-performance-stop-measuring',
      // The sidebar's running time and the card's measuring time each tick once a second.
      budget: { idleUpdates: 40, idleCommits: 20 },
      setup: async (p) => {
        await perfStubs(p, true)
        await reloadUnder(p, tabs)
        await openGame(p, tabs(p), 'Stardew Valley')
        await tab(p, 'Performance')
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Start measuring' }).click()
        await p.getByRole('button', { name: 'Stop measuring' }).waitFor()
      },
      teardown: async (p) => {
        await perfStubs(p, false)
        await reloadUnder(p, tabs)
      },
    },
    overlayFieldsCase(tabs, tab),
    {
      name: 'game-page-no-load-order',
      setup: async (p) => {
        await loadOrderStubs(p, true)
        await reloadUnder(p, tabs)
      },
      run: async (p) => {
        await tab(p, 'Home')
        if (
          (await tabs(p)
            .getByRole('tab', { name: /^Load order/ })
            .count()) > 0
        ) {
          throw new Error('a game whose loader has no load order still shows the Load order tab')
        }
      },
      teardown: async (p) => {
        await loadOrderStubs(p, false)
        await reloadUnder(p, tabs)
      },
    },
  ]
}
