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
  // A profile whose game has mod projects in the Develop tab.
  developing: { game: string; name: string }
  stepMs: number
}

/** The news posts' scenes: 0.3.13's Change version, Add a game and Multiplayer mod list, and 0.3.14's Develop tab and
 * Preferred source. */
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
  {
    name: 'develop',
    ...h.desk,
    profile: h.developing,
    steps: async (page) => {
      await h.click(page, 'tab', 'Develop')
      await page.getByRole('button', { name: 'Game code…' }).waitFor({ timeout: h.stepMs })
      // Each card's icon arrives with the project's details, read after the list shows.
      await page.waitForFunction(
        () => document.querySelectorAll('main img').length >= 5,
        undefined,
        {
          timeout: h.stepMs,
        },
      )
    },
  },
  {
    name: 'source-order',
    ...h.desk,
    profile: h.profile,
    steps: async (page) => {
      await h.click(page, 'tab', 'Home')
      await page
        .getByRole('button', { name: / settings$/u })
        .last()
        .click({ timeout: h.stepMs })
      await page.getByRole('button', { name: /^All .* profiles$/u }).click({ timeout: h.stepMs })
      await page.getByRole('button', { name: 'Mods', exact: true }).click({ timeout: h.stepMs })
      await page
        .getByText('Preferred source', { exact: true })
        .evaluate((el) => el.closest('.settings-tiles > *')?.scrollIntoView({ block: 'start' }))
      // One source off, so the picture shows what the switches do (the sandbox's own setting).
      const curseforge = page.getByRole('switch', { name: 'Search CurseForge' })
      if (await curseforge.isChecked()) {
        await curseforge.click({ timeout: h.stepMs })
      }
    },
  },
]
