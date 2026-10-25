import type { Page } from '@playwright/test'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'
import { VIEWPORT } from './renderViewport.ts'

const SHORT_WINDOW = { width: 1280, height: 600 }

const SETTINGS_SECTIONS = [
  'General',
  'Appearance',
  'Mods and profiles',
  'Downloads',
  'Accounts',
  'Source health',
  'Updates',
  'Notifications',
  'Storage',
  'Launchers',
  'Shortcuts',
  'About',
]

/** Screens no interaction case reaches, so every one is measured, checked for layout breaks and pictured. */
export function screenCases(h: Helpers): Case[] {
  const { toWorkspace, tab } = h
  // Browse has its own view buttons; the shared view() helper switches the Mods tab.
  const browseView = (p: Page, which: 'List view' | 'Grid view') =>
    p.getByRole('button', { name: which }).first().click()
  const section = (name: string): Case => ({
    name: `settings-${name.toLowerCase().replaceAll(' ', '-')}`,
    setup: async (p: Page) => {
      await toWorkspace(p)
      await openSettings(p)
    },
    // The title bar's bell is also named Notifications, so the section is looked up in the settings navigation.
    run: (p) =>
      p
        .getByRole('navigation', { name: 'Settings sections' })
        .getByRole('button', { name, exact: true })
        .click(),
  })
  return [
    {
      name: 'problems-fixes',
      setup: (p) => tab(p, 'Problems'),
      // The first row's other fixes; a profile without problems has no menu, and the case only opens the tab.
      run: async (p) => {
        const arrow = p.getByRole('button', { name: 'Other fixes' }).first()
        if ((await arrow.count()) > 0) {
          await arrow.click()
        }
      },
      teardown: (p) => p.keyboard.press('Escape'),
    },
    {
      name: 'browse-list',
      // A list row lays out differently from a grid card, so every result card renders again, and a list screen holds
      // every row its height fits.
      budget: { updates: 2600 },
      setup: (p) => tab(p, 'Browse'),
      run: (p) => browseView(p, 'List view'),
    },
    {
      name: 'browse-details',
      setup: async (p) => {
        await tab(p, 'Browse')
        await browseView(p, 'Grid view')
      },
      run: (p) =>
        p
          .getByRole('img', { name: /endorsements|downloads/ })
          .first()
          .click(),
    },
    { name: 'downloads-sheet', setup: toWorkspace, run: (p) => p.keyboard.press('Control+j') },
    {
      // A short window: the sidebar's sections and Play all fit, with nothing to scroll.
      name: 'short-window',
      setup: (p) => tab(p, 'Home'),
      run: async (p) => {
        await p.setViewportSize(SHORT_WINDOW)
        const list = p.getByRole('tablist', { name: 'Profile sections' })
        // The resize lays out on the next frames; measuring before then reads the old heights.
        await list.evaluate(
          () => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))),
        )
        const [needs, has, item, inner] = await list.evaluate((el) => [
          el.scrollHeight,
          el.clientHeight,
          el.querySelector('[role="tab"]')?.getBoundingClientRect().height ?? 0,
          globalThis.innerHeight,
        ])
        if (needs > has) {
          throw new Error(
            `the sidebar scrolls in a short window: ${needs}px in ${has}px (items ${item}px, window ${inner}px)`,
          )
        }
      },
      teardown: (p) => p.setViewportSize(VIEWPORT),
    },
    {
      name: 'develop-new-mod',
      setup: (p) => tab(p, 'Develop'),
      run: (p) => p.getByRole('button', { name: 'New mod…' }).click(),
    },
    ...SETTINGS_SECTIONS.map(section),
  ]
}
