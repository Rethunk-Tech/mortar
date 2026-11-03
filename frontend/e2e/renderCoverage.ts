import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import process from 'node:process'
import type { Page, Route } from '@playwright/test'
import type { Result } from '../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'
import { openGame, otherGameIds, returnToMainGame } from './renderProfile.ts'

const SEARCHABLE_SOURCES = 2_448_717_962
const PROBLEMS = 2_301_867_050
const PROBLEMS_NO_WAIT = 1_691_579_924
const TABS = ['Home', 'Mods', 'Config', 'Problems', 'Browse', 'Develop']

/** Cards and rows of the Mods tab and Browse open the same context menu by a right-click. */
function menuCases({ tab, view, row }: Helpers): Case[] {
  const menu = (p: Page) => p.getByRole('menu').getByRole('menuitem').first().waitFor()
  const close = (p: Page) => p.keyboard.press('Escape')
  const modMenu = (name: string, which: 'List view' | 'Grid view'): Case => ({
    name,
    setup: (p) => view(p, which),
    run: async (p) => {
      await row(p, 0).click({ button: 'right' })
      await menu(p)
    },
    teardown: close,
  })
  let results = false
  return [
    modMenu('mods-card-menu', 'Grid view'),
    modMenu('mods-list-menu', 'List view'),
    {
      name: 'browse-card-menu',
      setup: async (p) => {
        await tab(p, 'Browse')
        // Results come from the network, so the case measures nothing when none arrived.
        results = await p
          .locator('[data-hit]')
          .first()
          .waitFor({ timeout: 8000 })
          .then(() => true)
          .catch(() => false)
      },
      run: async (p) => {
        if (results) {
          await p.locator('[data-hit]').first().click({ button: 'right' })
          await menu(p)
        }
      },
      teardown: close,
    },
  ]
}

/** Answers the named binding calls with the server's own answer after `edit` has changed it; others go on untouched. */
function patchCalls<T>(ids: number[], edit: (answer: T) => T) {
  return async (route: Route) => {
    const body = route.request().postData() ?? ''
    if (!ids.some((id) => body.includes(`"methodID":${id}`))) {
      await route.fallback()
      return
    }
    const real = await route.fetch()
    await route.fulfill({ response: real, json: edit((await real.json()) as T) })
  }
}

/** Runs `edit` on the page's calls for as long as the case's screen is up. */
function patchedCase(
  c: Pick<Case, 'name' | 'run' | 'budget'> & { start: (p: Page) => Promise<void> },
  handler: (route: Route) => Promise<void>,
): Case {
  return {
    name: c.name,
    setup: async (p) => {
      await p.route('**/wails/runtime', handler)
      await c.start(p)
    },
    run: c.run,
    budget: c.budget,
    teardown: (p) => p.unroute('**/wails/runtime', handler),
  }
}

/** Browse with one of the game's sources switched off, which leaves its button out of the source row. */
function browseSourcesCase({ tab }: Helpers): Case {
  const lastOff = patchCalls<{ off: boolean }[] | null>([SEARCHABLE_SOURCES], (sources) => {
    const last = sources && sources.length > 1 ? sources.at(-1) : undefined
    if (last) {
      last.off = true
    }
    return sources
  })
  return patchedCase(
    {
      name: 'browse-sources',
      start: (p) => tab(p, 'Home'),
      run: async (p) => {
        await tab(p, 'Browse')
        await p.getByRole('group', { name: 'Source' }).waitFor()
      },
    },
    lastOff,
  )
}

const mod = (name: string) => `smapi:Render.${name}`

/** One finding of nearly every kind the Problems tab sorts into a section, added to the profile's own result. */
function withFindings(result: Result): Result {
  const ref = { site: 'nexus', github: '' }
  const copy = (version: string, newest: boolean) => ({
    key: `Dup-${version}`,
    name: 'Render Duplicate',
    version,
    source: 'nexus',
    nexus: true,
    newest,
    needed: null,
    tooOld: null,
  })
  return {
    ...result,
    missing: [
      ...(result.missing ?? []),
      {
        dependentId: mod('Needy'),
        dependentName: 'Render Needy',
        id: mod('Absent'),
        minimumVersion: '1.0.0',
        reason: '',
        installedVersion: '',
        listed: false,
        note: '',
        optional: false,
        where: ref,
      },
    ],
    duplicates: [
      ...(result.duplicates ?? []),
      {
        id: mod('Duplicate'),
        name: 'Render Duplicate',
        copies: [copy('1.0', false), copy('2.0', true)],
      },
    ],
    broken: [
      ...(result.broken ?? []),
      {
        key: 'Broken',
        id: mod('Broken'),
        name: 'Render Broken',
        status: 'broken',
        brokeIn: '1.6.0',
        summary: 'Crashes on load since the game update',
      },
    ],
    cleanup: [...(result.cleanup ?? []), { key: 'Stale', id: mod('Stale'), name: 'Render Stale' }],
    damaged: [
      ...(result.damaged ?? []),
      {
        key: 'Damaged',
        name: 'Render Damaged',
        missing: 1,
        changed: 2,
        extra: 0,
        files: ['a.dll'],
      },
    ],
    runErrors: [
      ...(result.runErrors ?? []),
      {
        key: 'Noisy',
        id: mod('Noisy'),
        name: 'Render Noisy',
        count: 3,
        start: 'NullReferenceException',
        severe: true,
        runId: '',
      },
    ],
  }
}

function problemsSectionsCase({ tab }: Helpers): Case {
  // The profile's problems are already loaded; a window focus makes the app scan again, now through the patch.
  const rescan = (p: Page) => p.evaluate(() => globalThis.dispatchEvent(new Event('focus')))
  const patched = patchedCase(
    {
      name: 'problems-sections',
      // Every section at once: each row mounts its fix button and menu, and the sections lay out again as each lands.
      budget: { updates: 2600 },
      start: async (p) => {
        await tab(p, 'Home')
        await rescan(p)
      },
      run: async (p) => {
        await tab(p, 'Problems')
        await p.getByText('Render Needy').first().waitFor()
      },
    },
    patchCalls<Result>([PROBLEMS, PROBLEMS_NO_WAIT], withFindings),
  )
  return {
    ...patched,
    teardown: async (p) => {
      await patched.teardown?.(p)
      await rescan(p)
    },
  }
}

/** The catalog's name for each game id, which the title bar's game menu lists. */
function gameNames(): Map<string, string> {
  const catalog = JSON.parse(
    readFileSync(new URL('../../components.json', import.meta.url), 'utf8'),
  ) as { games: { id: string; name: string }[] }
  return new Map(catalog.games.map((g) => [g.id, g.name]))
}

/** The switch, and the Mods tab in whichever view the last case left. */
function budgetFor(tab: string): Pick<Case, 'budget'> {
  if (tab === 'Home') {
    // Both games hold a profile: the switch redraws the old profile's workspace and mounts the new one's, which costs
    // more than a setup screen.
    return { budget: { updates: 2000 } }
  }
  // A grid mounts more cards than a list rows, the view the cases before left is remembered, and the copied data decides
  // how many mods there are.
  return tab === 'Mods' ? { budget: { updates: 2000 } } : {}
}

/**
 * Every other game with a profile in the copied data: the switch to it, then each of its main tabs. Generated from the
 * data, so a new game needs no code; the seeded sandbox holds one game, and a data folder with one game has no other.
 */
function otherGameCases({ tab, tabs }: Helpers): Case[] {
  const dataDir =
    process.env.MORTAR_RENDER_DATA ?? join(process.env.HOME ?? '', '.local', 'share', 'mortar')
  const ids = process.env.MORTAR_RENDER_SEEDED ? [] : otherGameIds(dataDir)
  if (ids.length === 0) {
    console.log('other-game cases skipped: the render data holds one game')
    return []
  }
  const names = gameNames()
  const here = (name: string) => (p: Page) => openGame(p, tabs(p), name)
  return ids.flatMap((id) => {
    const name = names.get(id) ?? id
    return TABS.map((t, i): Case => {
      const last = i === TABS.length - 1
      return {
        name: `visit-${id}-${t.toLowerCase()}`,
        ...budgetFor(t),
        setup: i === 0 ? (p) => tab(p, 'Home') : here(name),
        run:
          i === 0
            ? here(name)
            : async (p) => {
                // Only a loader with a devTemplate has the Develop tab.
                const there = await tabs(p)
                  .getByRole('tab', { name: new RegExp(`^${t}`) })
                  .count()
                if (t !== 'Develop' || there > 0) {
                  await tab(p, t)
                }
              },
        // The cases after expect the run's own game.
        ...(last
          ? {
              teardown: async (p: Page) => {
                await returnToMainGame(p, tabs(p), async () => 0)
              },
            }
          : {}),
      }
    })
  })
}

/** Cases for the menus, the source row and the games a run does not open on its own. */
/** The account cards below the fold, which the Accounts section's own picture cuts off. */
const modioAccountCase: Case = {
  name: 'settings-accounts-modio',
  setup: openSettings,
  run: async (p) => {
    await p
      .getByRole('navigation', { name: 'Settings sections' })
      .getByRole('button', { name: 'Accounts', exact: true })
      .click()
    const card = p.getByText('mod.io', { exact: true }).first()
    await card.waitFor()
    await card.evaluate((el) => el.scrollIntoView({ block: 'center' }))
  },
}

/** The editors the setting finds, which General's own picture shows closed. */
const codeEditorCase: Case = {
  name: 'settings-code-editor',
  setup: openSettings,
  run: async (p) => {
    const select = p.getByRole('combobox', { name: 'Code editor' })
    await select.scrollIntoViewIfNeeded()
    await select.click()
    await p.getByRole('option', { name: 'Other program…' }).waitFor()
  },
}

export function coverageCases(h: Helpers): Case[] {
  return [
    modioAccountCase,
    codeEditorCase,
    ...menuCases(h),
    browseSourcesCase(h),
    problemsSectionsCase(h),
    ...otherGameCases(h),
  ]
}
