import type { Page, Route } from '@playwright/test'
import type { Case } from './render-budget.ts'
import { patchCalls } from './renderCoverage.ts'
import type { dialogKit } from './renderDialogs.ts'
import { openGame } from './renderProfile.ts'

const GET_SETTINGS = 2_353_745_283
const LIST_PROFILES = 3_126_696_187
const LOADER_INSTALL = 925_441_952
const LOADER_LOCAL_STATUS = 4_217_188_234
const LOADER_STATUS = 607_125_999
const PROFILE_MODS = 3_651_195_110

const STARDEW = 'stardew'

type Stub = (route: Route) => Promise<void>

interface LoaderState {
  installed: boolean
  broken: boolean
  shared: boolean
  updateAvailable: boolean
  version: string
  latest: string
}
interface ModRow extends Record<string, unknown> {
  key: string
  id: string
  name: string
  enabled: boolean
  needs?: string[]
}

/** Answers the loader's install call with a refusal, so a case never writes SMAPI into the copied game folder. */
const installRefused: Stub = async (route) => {
  if (!(route.request().postData() ?? '').includes(`"methodID":${LOADER_INSTALL}`)) {
    await route.fallback()
    return
  }
  await route.fulfill({
    status: 500,
    contentType: 'application/json',
    body: JSON.stringify({
      kind: 'RuntimeError',
      message: 'could not reach the release server',
      cause: { kind: 'network' },
    }),
  })
}

const loaderMissing = patchCalls<LoaderState>([LOADER_LOCAL_STATUS, LOADER_STATUS], (s) => ({
  ...s,
  installed: false,
  broken: false,
  shared: false,
  updateAvailable: false,
  version: '',
  latest: '4.1.10',
}))

const noProfiles = patchCalls<unknown[] | null>([LIST_PROFILES], () => [])

const launchersUnconfirmed = patchCalls<{ launchersConfirmed: boolean }>([GET_SETTINGS], (s) => ({
  ...s,
  launchersConfirmed: false,
}))

/**
 * Three mods around the first listed one, each in its own download: Render Main needs the Render Helper library
 * (which came in beside Render Core, so it reads implicit), and Render Dependent needs Render Main.
 */
const withDisableChain = patchCalls<ModRow[] | null>([PROFILE_MODS], (mods) => {
  const base = mods?.[0]
  if (!(mods && base)) {
    return mods
  }
  const row = (key: string, name: string, needs: string[]): ModRow => ({
    ...base,
    key,
    id: `smapi:Render.${name}`,
    name: `Render ${name}`,
    enabled: true,
    needs,
    optional: [],
    contentPackFor: '',
  })
  return [
    ...mods,
    row('render-core-download', 'Core', []),
    { ...row('render-core-download', 'Helper', ['smapi:Render.Core']) },
    row('render-main-download', 'Main', ['smapi:Render.Helper']),
    row('render-dependent-download', 'Dependent', ['smapi:Render.Main']),
  ]
})

function emit(p: Page, events: { name: string; data: unknown }[]) {
  return p.evaluate((list) => {
    for (const e of list) {
      ;(
        globalThis as unknown as {
          _wails: { dispatchWailsEvent: (e: { name: string; data: unknown }) => void }
        }
      )._wails.dispatchWailsEvent(e)
    }
  }, events)
}

interface Kit {
  reload: (p: Page) => Promise<void>
  under: (...stubs: Stub[]) => (p: Page, on: boolean) => Promise<void>
  stardew: (p: Page) => Promise<void>
  tab: (p: Page, name: string) => Promise<void>
}

function firstRunCases({ reload, under }: Kit): Case[] {
  const firstRunStubs = under(launchersUnconfirmed)
  const gameSetupStubs = under(noProfiles, loaderMissing, installRefused)
  return [
    {
      name: 'first-run-launchers',
      setup: async (p) => {
        await firstRunStubs(p, true)
        await reload(p)
      },
      run: async (p) => {
        await p.getByRole('heading', { name: 'Welcome to Mortar' }).waitFor()
        await p.getByRole('button', { name: /^Join our / }).waitFor()
      },
      teardown: async (p) => {
        await firstRunStubs(p, false)
        await reload(p)
      },
    },
    {
      name: 'first-run-loader',
      setup: async (p) => {
        await gameSetupStubs(p, true)
        await reload(p)
        await p
          .getByRole('button', { name: /^(Open|Set up) Stardew Valley/ })
          .first()
          .click()
        await p.getByRole('heading', { name: /^Set up Stardew Valley/ }).waitFor()
      },
      run: async (p) => {
        const step = p.getByRole('heading', { name: /^Installing / })
        // The game folder step, then Nexus's, lead to the loader step.
        for (let n = 0; n < 3 && !(await step.isVisible()); n++) {
          await p
            .getByRole('button', { name: /^(Continue|Skip for now)$/ })
            .first()
            .click()
          await step.waitFor({ timeout: 2000 }).catch(() => undefined)
        }
        await step.waitFor()
        await p.getByRole('alert').first().waitFor()
      },
      teardown: async (p) => {
        await gameSetupStubs(p, false)
        await reload(p)
      },
    },
  ]
}

function disableCases({ reload, under, stardew, tab }: Kit): Case[] {
  const disableStubs = under(withDisableChain)
  return [
    {
      name: 'mods-disable-also',
      setup: async (p) => {
        await disableStubs(p, true)
        await reload(p)
        await stardew(p)
        await tab(p, 'Mods')
        await p.getByRole('button', { name: 'List view' }).click()
        await p.getByRole('textbox', { name: /^Filter / }).fill('Render Main')
        await p.getByRole('switch', { name: 'Disable Render Main' }).waitFor()
      },
      run: async (p) => {
        await p.getByRole('switch', { name: 'Disable Render Main' }).click()
        const d = p.getByRole('dialog').last()
        await d.getByRole('checkbox', { name: /^Also disable 1 mod that needs it/ }).waitFor()
        await d
          .getByRole('checkbox', { name: /^Also disable 1 library nothing else needs/ })
          .waitFor()
      },
      teardown: async (p) => {
        await p.getByRole('dialog').last().getByRole('button', { name: 'Cancel' }).click()
        await p.getByRole('textbox', { name: /^Filter / }).fill('')
        await disableStubs(p, false)
        await reload(p)
      },
    },
  ]
}

function loaderCases({ reload, under, stardew, tab }: Kit): Case[] {
  const loaderStubs = under(loaderMissing)
  return [
    {
      name: 'loader-banner-missing',
      setup: async (p) => {
        await loaderStubs(p, true)
        await reload(p)
        await stardew(p)
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p
          .getByRole('status')
          .getByText(/ is not installed$/)
          .waitFor()
        await p.getByRole('status').getByRole('button', { name: 'Install' }).waitFor()
      },
      teardown: async (p) => {
        await loaderStubs(p, false)
        await reload(p)
      },
    },
    {
      name: 'loader-banner-installing',
      setup: async (p) => {
        await loaderStubs(p, true)
        await reload(p)
        await stardew(p)
        await tab(p, 'Home')
        await p.getByRole('status').getByRole('button', { name: 'Install' }).waitFor()
      },
      run: async (p) => {
        await emit(p, [
          { name: 'loader:state', data: { game: STARDEW, installing: true, error: '' } },
          { name: 'loader:progress', data: { step: 'downloaded' } },
        ])
        await p.getByRole('status').getByText('Downloaded').waitFor()
      },
      teardown: async (p) => {
        // The reload drops the install state; an event that ended it would start a status call mid-reload.
        await loaderStubs(p, false)
        await reload(p)
      },
    },
    {
      name: 'game-settings-loader-missing',
      setup: async (p) => {
        await loaderStubs(p, true)
        await reload(p)
        await stardew(p)
        await tab(p, 'Home')
        await p
          .getByRole('button', { name: / settings$/ })
          .last()
          .click()
        await p.getByRole('button', { name: /^This profile: / }).waitFor()
        await p
          .getByRole('navigation', { name: 'Settings sections' })
          .getByRole('button', { name: 'SMAPI', exact: true })
          .click()
      },
      run: async (p) => {
        await p
          .getByText(/ is not installed$/)
          .first()
          .waitFor()
        await p
          .getByRole('button', { name: /^Install/ })
          .first()
          .waitFor()
      },
      teardown: async (p) => {
        await loaderStubs(p, false)
        await reload(p)
      },
    },
  ]
}

/**
 * The first-run flow, the Disable dialog's two also-disable rows and the loader's install screens, each under the
 * binding answers that put the copied data in the state that shows them.
 */
export function flowCases(
  { ready }: ReturnType<typeof dialogKit>,
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case[] {
  const kit: Kit = {
    tab,
    reload: async (p) => {
      await p.reload()
      await tabs(p)
        .or(p.getByRole('heading', { name: /^Welcome to/ }))
        .or(p.getByRole('button', { name: /^(Open|Set up) / }).first())
        .first()
        .waitFor({ timeout: 30_000 })
    },
    under:
      (...stubs) =>
      async (p, on) => {
        for (const s of stubs) {
          await (on ? p.route('**/wails/runtime', s) : p.unroute('**/wails/runtime', s))
        }
      },
    stardew: async (p) => {
      await ready(p)
      await openGame(p, tabs(p), 'Stardew Valley')
    },
  }
  return [...firstRunCases(kit), ...disableCases(kit), ...loaderCases(kit)]
}
