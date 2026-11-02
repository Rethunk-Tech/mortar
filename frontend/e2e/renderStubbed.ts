import type { Page, Route } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'

// Backend answers the render data may not produce on its own (a Steam install, a project out of step, a GitHub origin).
const CAN_GET_OLDER_VERSION = 3_208_769_223
const OLDER_VERSION_DEPOTS = 1_490_403_490
const DEPENDENCY_SYNC = 458_110_213
const UPLOAD_SITES = 858_225_634
const LOAD_ORDER = 3_963_816_757
const PACK = 1_330_956_574
const UPLOAD_GITHUB = 3_405_360_245

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

// Release everywhere: the package is stubbed as built, GitHub uploads, Thunderstore is refused and Nexus is not ready.
function releaseEverywhereCase({ developTab, pick }: ReturnType<typeof dialogKit>): Case {
  const stub = stubCalls({
    [UPLOAD_SITES]: [
      { id: 'thunderstore', configured: true, ready: true },
      { id: 'nexus', configured: true, ready: false },
      { id: 'github', configured: true, ready: true },
    ],
    [PACK]: { zip: '/tmp/render.zip', issues: [] },
    [UPLOAD_GITHUB]: 'https://github.com/example/mod/releases/tag/v1.0.0',
  })
  return {
    name: 'develop-release-everywhere',
    setup: async (p: Page) => {
      await p.route('**/wails/runtime', stub)
      await developTab(p)
      await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
      if ((await p.getByRole('menuitem', { name: /^Upload v/ }).count()) === 0) {
        throw new Error(
          'the first Develop project offers no Upload; the render data needs a released project',
        )
      }
    },
    run: async (p: Page) => {
      await pick(p, /^Upload v/)
      const d = p.getByRole('dialog').last()
      await d.getByRole('button', { name: 'Release everywhere' }).click()
      await d.getByText('Skipped: No credentials yet').waitFor()
      await d.getByRole('button', { name: 'Open page' }).waitFor()
    },
    teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
  }
}

/** The Load order tab as Baldur's Gate 3 shows it: GustavX, then paks in the profile's order, one with an issue. */
function loadOrderCase(tab: (page: Page, name: string) => Promise<void>): Case {
  const row = (position: number, name: string, over: Record<string, unknown> = {}) => ({
    position,
    id: `folder:uuid-${position}`,
    name,
    version: '1.0.0.0',
    detail: `${name}.pak`,
    ...over,
  })
  const stub = stubCalls({
    [LOAD_ORDER]: [
      row(1, 'GustavX', { official: true, detail: undefined }),
      row(2, 'Improved UI', { entry: 'improved-ui' }),
      row(3, 'Party Limit Begone', {
        entry: 'party-limit',
        issues: [{ code: 'moduleMissing', message: 'requires Gone', params: { other: 'Gone' } }],
      }),
      row(4, 'Hand-added override'),
    ],
  })
  return {
    name: 'load-order-bg3',
    setup: async (p: Page) => {
      await p.route('**/wails/runtime', stub)
      await tab(p, 'Home')
    },
    run: async (p: Page) => {
      await tab(p, 'Load order')
      await p.getByText('Party Limit Begone', { exact: true }).waitFor()
    },
    teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
  }
}

/** Get an older version with builds Mortar has seen, and the select that lists them open. */
function olderVersionBuildsCase({
  dialog,
  inDialog,
  tabIn,
  editProfile,
}: ReturnType<typeof dialogKit>): Case {
  const stub = stubCalls({
    [CAN_GET_OLDER_VERSION]: true,
    [OLDER_VERSION_DEPOTS]: {
      appId: '1966720',
      depots: [{ id: '1966721', manifest: '7081337227611034738', size: 1 }],
      builds: [
        {
          depotId: '1966721',
          manifest: '7081337227611034738',
          buildId: '15012345',
          seen: '2026-10-28',
          current: true,
        },
        {
          depotId: '1966721',
          manifest: '3322110099887766554',
          buildId: '14998877',
          seen: '2026-09-02',
          current: false,
        },
        { depotId: '1966721', manifest: '1122334455667788990', seen: '2026-07-15', current: false },
      ],
    },
  })
  const base = dialog(
    'older-version-builds',
    (p) => inDialog(p, /^Get an older version…/),
    undefined,
    async (p) => {
      await p.route('**/wails/runtime', stub)
      await editProfile(p)
      await tabIn(p, 'Launch')
      await p.getByRole('button', { name: /^Get an older version…/ }).waitFor()
    },
  )
  return {
    ...base,
    run: async (p: Page) => {
      await base.run(p)
      // An open menu hides the dialog from the accessibility tree, so the select opens last.
      await p.getByLabel('Builds Mortar has seen').click()
      await p.getByRole('option').first().waitFor()
    },
    teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
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
    tab,
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
            await p.getByRole('heading', { name: /^Out of step/ }).waitFor()
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
    releaseEverywhereCase(kit),
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
    loadOrderCase(tab),
    olderVersionBuildsCase(kit),
  ]
}
