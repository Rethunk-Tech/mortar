import type { Page } from '@playwright/test'

interface Helpers {
  click: (
    page: Page,
    role: Parameters<Page['getByRole']>[0],
    name: string | RegExp,
  ) => Promise<void>
  openGameSelect: (page: Page) => Promise<void>
  desk: { width: number; height: number }
  wide: { width: number; height: number }
  profile: { game: string; name: string }
  stepMs: number
}

/** The 0.3.13 news post's scenes: Change version, Add a game and the Multiplayer mod list. */
export const releaseScenes = (h: Helpers) => [
  {
    name: 'change-version',
    ...h.desk,
    profile: h.profile,
    steps: async (page) => {
      await h.click(page, 'tab', 'Mods')
      await page
        .getByRole('button', { name: /^Details of /u })
        .nth(5)
        .click({ timeout: h.stepMs })
      await h.click(page, 'button', /^Change version…/u)
      await page
        .getByRole('dialog')
        .getByRole('list', { name: 'Versions' })
        .waitFor({ timeout: 20_000 })
    },
  },
  {
    name: 'add-game',
    ...h.desk,
    steps: async (page) => {
      await h.openGameSelect(page)
      await h.click(page, 'button', /^Add a game…/u)
      await page.getByRole('dialog').waitFor({ timeout: h.stepMs })
    },
  },
  {
    name: 'multiplayer',
    ...h.wide,
    profile: h.profile,
    steps: async (page) => {
      await page
        .getByRole('button', { name: /^Switch profile/u })
        .click({ button: 'right', timeout: h.stepMs })
      await h.click(page, 'menuitem', /^Multiplayer mod list…/u)
      await page.getByRole('dialog').waitFor({ timeout: h.stepMs })
    },
  },
]
