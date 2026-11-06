import type { Page } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { dialogKit } from './renderDialogs.ts'
import { stubCalls, UPLOAD_SITES } from './renderStubbed.ts'

/** The Upload dialog with every site's row: Thunderstore names the team it uploads as, the others have no account. */
export function uploadDialogCases({ developTab, pick }: ReturnType<typeof dialogKit>): Case[] {
  const stub = stubCalls({
    [UPLOAD_SITES]: [
      { id: 'thunderstore', configured: true, ready: true, account: 'Render Team' },
      { id: 'github', configured: true, ready: true },
      { id: 'nexus', configured: true, ready: false },
      { id: 'curseforge', configured: false, ready: false },
    ],
  })
  return [
    {
      name: 'upload-dialog-sites',
      setup: async (p: Page) => {
        await p.route('**/wails/runtime', stub)
        await developTab(p)
        await p.getByRole('main').getByRole('button', { name: 'More actions' }).first().click()
        // A project without a released version offers no Upload; a skipped case would pass measuring nothing.
        if ((await p.getByRole('menuitem', { name: /^Upload v/ }).count()) === 0) {
          throw new Error(
            'the first Develop project offers no Upload; the render data needs a released project',
          )
        }
      },
      run: async (p: Page) => {
        await pick(p, /^Upload v/)
        await p
          .getByRole('dialog')
          .last()
          .getByText('Thunderstore as Render Team', { exact: true })
          .waitFor()
      },
      teardown: (p: Page) => p.unroute('**/wails/runtime', stub),
    },
  ]
}
