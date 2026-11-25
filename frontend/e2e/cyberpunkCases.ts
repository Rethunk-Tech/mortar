import type { Page } from '@playwright/test'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'
import { routeGameAs } from './renderCoverage.ts'
import { dropFakeQueue } from './renderMeter.ts'
import { biggestProfileIds, renderDataDir } from './renderProfile.ts'
import { type Stub, stubCalls } from './renderStubbed.ts'

// Binding method ids of the calls the Cyberpunk 2077 screens make (frontend/bindings).
const TOOLS_LIST = 903_228_865
const REDMOD_CHOICE = 749_032_213

const GB = 1024 ** 3

const failedDownload = (profileId: string) => ({
  items: [
    {
      id: 'render-active',
      kind: 'install',
      game: 'stardew',
      profileId,
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
      profileId,
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
})

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
  const convertStub = stubCalls({ [REDMOD_CHOICE]: 'convert' })
  let cyberpunk: Stub | undefined
  const asCyberpunk = async (p: Page) => {
    cyberpunk = await routeGameAs(p, 'cyberpunk-2077', { layout: 'cyberpunk' })
  }
  const profile = () => biggestProfileIds(renderDataDir())?.profile ?? ''
  const unroute = async (p: Page) => {
    for (const s of [toolsStub, cyberpunk, convertStub]) {
      if (s) {
        await p.unroute('**/wails/runtime', s)
      }
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
        }, failedDownload(profile()))
        await p.getByRole('button', { name: /^Downloads:/ }).click()
        // The failed row says why it offers Install anyway.
        await p
          .getByText('Mortar has no layout for this mod (3 files)', { exact: false })
          .first()
          .waitFor()
        await p.getByRole('button', { name: 'Install anyway' }).click()
        await p.getByRole('dialog').last().getByText('weird/thing.dat').waitFor()
      },
      // The fake queue's failure toast and rows must not outlive the case.
      teardown: async (p) => {
        await p.getByRole('dialog').last().getByRole('button', { name: 'Cancel' }).click()
        await p.keyboard.press('Escape')
        await dropFakeQueue(p, 'dialog-unknown-layout teardown')
      },
    },
    {
      name: 'tools-menu-game-tools',
      setup: async (p) => {
        await p.route('**/wails/runtime', toolsStub)
        await asCyberpunk(p)
        await p.reload()
        await tab(p, 'Home')
      },
      run: async (p) => {
        await p.getByRole('button', { name: 'Tools', exact: true }).click()
        await p.getByRole('menuitem', { name: 'REDLauncher' }).waitFor()
      },
      teardown: async (p) => {
        await p.keyboard.press('Escape')
        await unroute(p)
        await p.reload()
      },
    },
    {
      name: 'mods-menu-convert-redmod',
      setup: async (p) => {
        await asCyberpunk(p)
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
