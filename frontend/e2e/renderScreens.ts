import type { Page } from '@playwright/test'
import { openSettings } from './app.ts'
import type { Case } from './render-budget.ts'
import type { Helpers } from './renderCases.ts'

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
      name: 'browse-list',
      // A list row lays out differently from a grid card, so every result card renders again.
      budget: { updates: 1200 },
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
          .getByText(/endorsements|downloads/)
          .first()
          .click(),
    },
    { name: 'downloads-sheet', setup: toWorkspace, run: (p) => p.keyboard.press('Control+j') },
    ...SETTINGS_SECTIONS.map(section),
  ]
}
