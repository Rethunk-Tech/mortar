import type { Page } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'
import { patchCalls } from './renderCoverage.ts'
import { stubCalls } from './renderStubbed.ts'

// Binding method ids of the calls the Cyberpunk 2077 screens make (frontend/bindings).
const GAME_LIST = 2_813_804_068
const TOOLS_LIST = 903_228_865
const REDMOD_CHOICE = 749_032_213

const GB = 1024 ** 3

const failedDownload = {
  items: [
    {
      id: 'render-active',
      kind: 'install',
      game: 'stardew',
      profileId: '',
      name: 'Render Active Pack',
      fileName: 'active.zip',
      state: 'downloading',
      sizeKb: 1024,
      progress: 40,
      speed: 1_000_000,
      error: '',
    },
    {
      id: 'render-no-layout',
      kind: 'install',
      game: 'stardew',
      profileId: '',
      name: 'Render Odd Mod',
      fileName: 'odd.zip',
      state: 'failed',
      sizeKb: 1024,
      progress: 100,
      speed: 0,
      partial: 0.1 * GB,
      error: 'Mortar has no layout for this mod (3 files), so it would place every file as packed',
      errorKind: 'layout',
      unknownLayout: {
        key: 'local-render',
        reason: 'unrecognised',
        files: ['weird/thing.dat', 'weird/other.dat', 'weird/readme.bin'],
        ask: true,
      },
    },
  ],
  paused: false,
  limitedUntil: 0,
}

const tool = (id: string, name: string) => ({
  id: `game:${id}`,
  name,
  executable: '',
  arguments: [],
  workingDir: '',
  builtin: true,
})

/** The screens Cyberpunk 2077 adds: the Install anyway prompt, the game's own tools and the REDmod conversion item. */
export function cyberpunkCases(h: Helpers): Case[] {
  const { tab, view, row, toWorkspace } = h
  const toolsStub = stubCalls({
    [TOOLS_LIST]: [
      tool('redlauncher', 'REDLauncher'),
      tool('redeploy', 'REDmod Deploy Latest Load Order'),
      tool('cybercat', 'CyberCAT Save Editor'),
    ],
  })
  // The open game answers as Cyberpunk 2077's layout would, so its mod menu offers the conversion, which the copied data's
  // game has no archive mod to offer it for.
  const cyberpunk = patchCalls<Record<string, unknown>[]>([GAME_LIST], (games) =>
    games.map((g) => ({ ...g, layout: 'cyberpunk' })),
  )
  const convertStub = stubCalls({ [REDMOD_CHOICE]: 'convert' })
  const stubs = [toolsStub, cyberpunk, convertStub]
  const unroute = async (p: Page) => {
    for (const s of stubs) {
      await p.unroute('**/wails/runtime', s)
    }
  }
  return [
    {
      name: 'dialog-unknown-layout',
      setup: toWorkspace,
      run: async (p) => {
        await p.evaluate((state) => {
          const w = globalThis as unknown as {
            _wails: { dispatchWailsEvent: (e: { name: string; data: unknown }) => void }
          }
          w._wails.dispatchWailsEvent({ name: 'queue:changed', data: state })
        }, failedDownload)
        await p.getByRole('button', { name: /^Downloads:/ }).click()
        await p.getByRole('button', { name: 'Install anyway' }).click()
        await p.getByRole('dialog').last().getByText('weird/thing.dat').waitFor()
      },
      teardown: async (p) => {
        await p.getByRole('dialog').last().getByRole('button', { name: 'Cancel' }).click()
        await p.keyboard.press('Escape')
      },
    },
    {
      name: 'tools-menu-game-tools',
      setup: async (p) => {
        await p.route('**/wails/runtime', toolsStub)
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Tools', exact: true }).click()
        await p.getByRole('menuitem', { name: 'REDLauncher' }).waitFor()
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
        await p.unroute('**/wails/runtime', toolsStub)
      },
    },
    {
      name: 'mods-menu-convert-redmod',
      setup: async (p) => {
        await p.route('**/wails/runtime', cyberpunk)
        await p.route('**/wails/runtime', convertStub)
        await p.reload()
        await view(p, 'List view')
      },
      run: async (p) => {
        await row(p, 0).click({ button: 'right' })
        await p.getByRole('menuitem', { name: 'Convert to REDmod' }).waitFor()
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
        await unroute(p)
        await p.reload()
      },
    },
  ]
}
