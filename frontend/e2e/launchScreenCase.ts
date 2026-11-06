import { join } from 'node:path'
import type { Page } from '@playwright/test'
import type { Case } from './render-budget.ts'
import { biggestProfileIds } from './renderProfile.ts'

type Wails = { _wails: { dispatchWailsEvent: (e: { name: string; data: unknown }) => void } }

function emit(p: Page, events: { name: string; data: unknown }[]) {
  return p.evaluate((list) => {
    for (const e of list) {
      ;(globalThis as unknown as Wails)._wails.dispatchWailsEvent(e)
    }
  }, events)
}

// The same data the run opened its profile from (render-budget.ts land).
const dataDir = () =>
  process.env.MORTAR_RENDER_DATA ?? join(process.env.HOME ?? '', '.local', 'share', 'mortar')

const status = (game: string, profile: string, state: string, over: object = {}) => ({
  name: 'launch:state',
  data: {
    game,
    profile,
    state,
    since: Date.now(),
    hint: '',
    error: '',
    ...over,
  },
})

const progress = (game: string, profile: string, over: object = {}) => ({
  name: 'launch:progress',
  data: {
    game,
    profile,
    via: 'steam',
    handed: false,
    copies: 1,
    up: 0,
    loader: 'smapi',
    tracks: true,
    readies: true,
    loaderStarted: false,
    loaded: 0,
    total: 0,
    mod: '',
    ready: false,
    usualMs: 24_000,
    ...over,
  },
})

/** The launch screen as the backend's launch events drive it: the game starting, its loader loading mods, and a failure. */
export function launchScreenCases(): Case[] {
  const ids = () => {
    const found = biggestProfileIds(dataDir())
    if (!found) {
      throw new Error('the render data holds no profile to launch')
    }
    return found
  }
  // A failed launch stays up until closed; the others end with the game closing.
  const end = async (p: Page) => {
    const { game, profile } = ids()
    await emit(p, [status(game, profile, 'idle')])
    const close = p.getByRole('button', { name: 'Close', exact: true })
    if (await close.isVisible()) {
      await close.click()
    }
    await p.getByRole('dialog', { name: /^Launching|did not start/ }).waitFor({ state: 'detached' })
  }
  // The marks' arcs and the elapsed clock tick while a launch waits, so its idle watch is never quiet.
  const ticking = { idleUpdates: 5500, idleCommits: 25 }
  return [
    {
      name: 'launch-screen-launching',
      budget: ticking,
      run: async (p) => {
        const { game, profile } = ids()
        await emit(p, [status(game, profile, 'launching'), progress(game, profile)])
        await p.getByRole('dialog', { name: /^Launching/ }).waitFor()
      },
      teardown: end,
    },
    {
      name: 'launch-screen-loading',
      // Opening the screen also switches the game page to its Console tab.
      budget: { ...ticking, updates: 1500 },
      run: async (p) => {
        const { game, profile } = ids()
        await emit(p, [
          status(game, profile, 'launching'),
          progress(game, profile, {
            handed: true,
            up: 1,
            loaderStarted: true,
            loaded: 12,
            total: 40,
            mod: 'Render Mod',
          }),
          status(game, profile, 'running'),
        ])
        await p.getByText('Loading 12 of 40 mods').waitFor()
      },
      teardown: end,
    },
    {
      name: 'launch-screen-failed',
      run: async (p) => {
        const { game, profile } = ids()
        await emit(p, [
          status(game, profile, 'launching'),
          progress(game, profile),
          status(game, profile, 'failed', { hint: 'steam', error: 'the game never wrote its log' }),
        ])
        await p
          .getByRole('heading', { name: /did not start/ })
          .first()
          .waitFor()
      },
      teardown: end,
    },
  ]
}
