import type { Page, Route } from '@playwright/test'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import { keyboardDragUp } from './renderCases.ts'
import { patchCalls } from './renderCoverage.ts'
import type { dialogKit } from './renderDialogs.ts'
import { biggestProfileIds, openGame, renderDataDir } from './renderProfile.ts'
import { stubCalls } from './renderStubbed.ts'

export const NEXUS_ACCOUNT = 1_189_232_250
const START_SSO = 305_010_372
const TRASH_PROJECT = 400_259_103
const GAME_LIST = 2_813_804_068
const LAUNCH_STATUS = 239_586_243
const MEASURE_IN_GAME = 1_064_106_242
const PEEK_IN_GAME = 512_110_599
const SANDBOX_REPORT = 3_359_515_706
const RUNS = 622_928_231
const RUN_LINES = 2_449_284_789

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

/** Reloads the app under the stubs, which the game list read at startup then goes through. */
export async function reloadUnder(p: Page, tabs: (p: Page) => ReturnType<Page['locator']>) {
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
  const running = patchCalls<{ state: string; profile: string }>([LAUNCH_STATUS], (status) => ({
    ...status,
    state: 'running',
    // Read per call: the seeded sandbox's data folder exists only once the run has started it.
    profile: biggestProfileIds(renderDataDir())?.profile ?? status.profile,
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
    configRunningCase(tabs, tab, running),
    developSortCase(developTab),
    sandboxedRunCase(tabs, tab),
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

/** Develop's Sort menu, then the Manual order's drag handles on the project cards. */
function developSortCase(developTab: (p: Page) => Promise<void>): Case {
  return {
    name: 'develop-sort-manual',
    // Two keyboard drags (there and back): each pick, move and drop redraws every card's sortable wrapper and grip.
    budget: { updates: 2000 },
    setup: developTab,
    run: async (p) => {
      await p.getByRole('button', { name: 'Sort projects' }).click()
      await p.getByRole('menuitemradio', { name: /^Manual/ }).click()
      await keyboardDragUp(p, p.getByRole('button', { name: /^Reorder / }), 1)
    },
  }
}

/** The Config page while the game runs: the fields stay editable under a warning that a change may wait. */
function configRunningCase(
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
  running: Stub,
): Case {
  return {
    name: 'config-game-running',
    // The sidebar's running time ticks once a second.
    budget: { idleUpdates: 40, idleCommits: 20 },
    setup: async (p) => {
      await p.route('**/wails/runtime', running)
      await reloadUnder(p, tabs)
      await openGame(p, tabs(p), 'Stardew Valley')
    },
    run: async (p) => {
      await tab(p, 'Config')
      await p.getByText(/^The game is running\. Most mods/).waitFor()
    },
    teardown: async (p) => {
      await p.unroute('**/wails/runtime', running)
      await reloadUnder(p, tabs)
    },
  }
}

/** A sandboxed run's Console note: the writable folders, and denied paths each with its Allow button. */
function sandboxedRunCase(
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case {
  // The run itself is stubbed too: the copied render data may hold no stored run for the profile, and the note only
  // shows under a run.
  const writable = [
    '/games/Stardew Valley',
    '/data/profiles/stardew/main',
    '/home/render/.config/StardewValley',
  ]
  const stub = stubCalls({
    [RUNS]: [
      {
        id: 'render-sandboxed',
        started: '2026-11-07T19:00:00Z',
        ended: '2026-11-07T19:05:00Z',
        durationMs: 300_000,
        loader: 'smapi',
        loaderVersion: '4.3.2',
        gameVersion: '1.6.15',
        outcome: 'ran',
        errors: 0,
        warnings: 1,
        sandbox: writable,
      },
    ],
    [RUN_LINES]: [
      {
        seq: 1,
        time: '19:00:01',
        level: 'WARN',
        mod: 'ExampleMod',
        message: 'Could not read /home/render/.config/ExampleMod/settings.json: permission denied',
        cont: false,
      },
    ],
    [SANDBOX_REPORT]: {
      writable,
      denied: [
        {
          path: '/home/render/.config/ExampleMod/settings.json',
          folder: '/home/render/.config/ExampleMod',
        },
        { path: '/home/render/.ssh/id_ed25519', folder: '' },
        { path: '/home/render/x.json', folder: '' },
      ],
      extra: '',
    },
  })
  return {
    name: 'console-sandboxed-run',
    setup: async (p) => {
      await p.route('**/wails/runtime', stub)
      await reloadUnder(p, tabs)
      await openGame(p, tabs(p), 'Stardew Valley')
      await tab(p, 'Console')
    },
    run: async (p) => {
      await p.getByText(/^This run was sandboxed: the game could write 3 folders/).waitFor()
      await p.getByText('The sandbox hid folders from the game or a mod').waitFor()
      await p.getByRole('button', { name: /^Allow \/home\/render\/\.config\/ExampleMod/ }).waitFor()
    },
    teardown: async (p) => {
      await p.unroute('**/wails/runtime', stub)
      await reloadUnder(p, tabs)
    },
  }
}
