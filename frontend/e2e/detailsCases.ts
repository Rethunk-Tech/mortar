import type { Page, Route } from '@playwright/test'
import type { Case } from './render-budget.ts'
import { patchCalls } from './renderCoverage.ts'
import { openGame } from './renderProfile.ts'
import { NEXUS_ACCOUNT, reloadUnder } from './stateCases.ts'

const CACHED_DETAILS = 504_566_085
const SAVED_AT = '2026-11-01T09:30:00Z'

const savedPage = (modId: number) => ({
  page: {
    modId,
    name: '',
    summary: 'Adds more vanilla-style crops, bushes, and trees to the game. Highly configurable.',
    description: '',
    pictureUrl: '',
    version: '1.7.0',
    author: '',
    uploadedBy: '',
    uploaderUrl: '',
    categoryId: 0,
    endorsements: 8169,
    endorsement: '',
    requirements: [],
    downloads: 776_700,
    uniqueDownloads: 0,
    created: '2024-03-01T00:00:00Z',
    updated: SAVED_AT,
    adult: false,
    status: 'published',
    available: true,
  },
  category: 'Crops',
  files: null,
  changelogs: null,
  stale: SAVED_AT,
})

/** The Mods details panel of a mod with an update, signed out of Nexus. */
export function detailsCases(
  ready: (p: Page) => Promise<void>,
  tabs: (p: Page) => ReturnType<Page['locator']>,
  tab: (p: Page, name: string) => Promise<void>,
): Case[] {
  const signedOut = patchCalls<{ signedIn: boolean; name: string; premium: boolean }>(
    [NEXUS_ACCOUNT],
    (account) => ({ ...account, signedIn: false, name: '', premium: false }),
  )
  // The render copy leaves out the cache, so each Nexus page the panel asks for is a saved one with a newer version.
  const savedPages = async (route: Route) => {
    const body = route.request().postData() ?? ''
    if (!body.includes(`"methodID":${CACHED_DETAILS}`)) {
      await route.fallback()
      return
    }
    const { args } = JSON.parse(body) as { args: { args: [string, number[] | null] } }
    const ids = args.args[1] ?? []
    // Only the panel's own read, so the cards keep their real numbers.
    const pages = ids.length === 1 ? { [String(ids[0])]: savedPage(ids[0] ?? 0) } : {}
    await route.fulfill({ contentType: 'application/json', body: JSON.stringify(pages) })
  }
  const panel = (p: Page) => p.getByRole('complementary', { name: 'Selected mod' })
  const openUpdated = async (p: Page) => {
    await openGame(p, tabs(p), 'Stardew Valley')
    await tab(p, 'Mods')
    await p
      .getByRole('button', { name: /^Details of Cornucopia/ })
      .first()
      .click()
    await panel(p).waitFor()
  }
  return [
    {
      // A mod with an update, signed out: the Nexus fields come from the saved page and say so.
      name: 'mods-details-nexus-saved',
      setup: async (p) => {
        await ready(p)
        await p.route('**/wails/runtime', signedOut)
        await p.route('**/wails/runtime', savedPages)
        await reloadUnder(p, tabs)
      },
      run: async (p) => {
        await openUpdated(p)
        await panel(p)
          .getByText(/Sign in to refresh it\.$/)
          .waitFor()
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
        await p.unroute('**/wails/runtime', signedOut)
        await p.unroute('**/wails/runtime', savedPages)
        await reloadUnder(p, tabs)
      },
    },
  ]
}
