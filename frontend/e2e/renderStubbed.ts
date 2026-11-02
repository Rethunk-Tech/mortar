import type { Page, Route } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'

// Backend answers the render data may not produce on its own (a Steam install, a project out of step, a GitHub origin).
const CAN_GET_OLDER_VERSION = 3_208_769_223
const DEPENDENCY_SYNC = 458_110_213
const UPLOAD_SITES = 858_225_634

type Stub = (route: Route) => Promise<void>
/** Answers the named binding calls with the given results; every other call goes on to the server. */
function stubCalls(answers: Record<number, unknown>): Stub {
  return async (route) => {
    const body = route.request().postData() ?? ''
    for (const [id, result] of Object.entries(answers)) {
      if (body.includes(`"methodID":${id}`)) {
        await route.fulfill({ contentType: 'application/json', body: JSON.stringify(result) })
        return
      }
    }
    await route.fallback()
  }
}

/** Screens the render data may not reach on its own, each with the backend answer that opens it stubbed. */
export function stubbedCases(kit: ReturnType<typeof dialogKit>): Case[] {
  const {
    dialog,
    pick,
    inDialog,
    tabIn,
    editProfile,
    developTab,
    share,
    toThunderstore,
    fromThunderstore,
  } = kit
  return [
    // The Mods tab with the project and its development profile disagreeing in all three ways.
    (() => {
      const stub = stubCalls({
        [DEPENDENCY_SYNC]: {
          profile: 'render-check',
          missing: [{ package: 'BepInEx-BepInExPack', version: '5.4.2100', soft: false }],
          undeclared: [{ package: 'Evaisa-LethalLib', version: '0.16.1' }],
          differs: [
            { package: 'Sigurd-CSync', declared: '5.0.0', installed: '5.0.1', soft: false },
          ],
        },
      })
      return {
        ...dialog(
          '',
          async (p) => {
            await tabIn(p, 'Mods')
            await p.getByText('Out of step', { exact: true }).waitFor()
          },
          undefined,
          async (p) => {
            await p.route('**/wails/runtime', stub)
            await developTab(p)
            await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
            await pick(p, /^Edit details…/)
          },
        ),
        name: 'develop-dependency-sync',
        teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
      }
    })(),
    // The upload dialog with GitHub among the sites; the menu offers Upload only for a project with a released version,
    // so the case skips itself when the render data's first project has none.
    (() => {
      const stub = stubCalls({
        [UPLOAD_SITES]: [
          { id: 'thunderstore', configured: true, ready: true },
          { id: 'github', configured: true, ready: true },
        ],
      })
      return {
        name: 'develop-upload-github',
        setup: async (p: Page) => {
          await p.route('**/wails/runtime', stub)
          await developTab(p)
          await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
          // The first project must have a released version for Upload to be offered; a case that skipped would
          // pass while measuring nothing.
          if ((await p.getByRole('menuitem', { name: /^Upload v/ }).count()) === 0) {
            throw new Error(
              'the first Develop project offers no Upload; the render data needs a released project',
            )
          }
        },
        run: async (p: Page) => {
          await pick(p, /^Upload v/)
          await p.getByRole('dialog').last().getByText('GitHub', { exact: true }).waitFor()
        },
        teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
      }
    })(),
    // The Launch tab offers the older version only for a Steam install, so the render sandbox's answer is stubbed.
    (() => {
      const stub = stubCalls({ [CAN_GET_OLDER_VERSION]: true })
      return {
        ...dialog(
          'older-version',
          (p) => inDialog(p, /^Get an older version…/),
          undefined,
          async (p) => {
            await p.route('**/wails/runtime', stub)
            await editProfile(p)
            await tabIn(p, 'Launch')
            await p.getByRole('button', { name: /^Get an older version…/ }).waitFor()
          },
        ),
        teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
      }
    })(),
    // The modpack form is open in setup: the case measures the upload dialog opening over it.
    {
      ...dialog(
        'upload-modpack',
        (p) => inDialog(p, /^Upload to Thunderstore…/),
        undefined,
        async (p) => {
          await toThunderstore(p)
          await share(p, /^Thunderstore/)
          await inDialog(p, /^Modpack/)
          await p.getByRole('button', { name: /^Upload to Thunderstore…/ }).waitFor()
        },
      ),
      teardown: fromThunderstore,
    },
  ]
}
