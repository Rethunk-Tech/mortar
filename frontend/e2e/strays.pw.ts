import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('the game Mods folder review moves the stray mod into the profile', async ({ page }) => {
  await openSeedFarm(page)
  await expect(
    page.getByText("1 mod in the game's Mods folder isn't in this profile"),
  ).toBeVisible()
  await page.getByRole('button', { name: 'Review…' }).click()
  const dialog = page.getByRole('dialog', { name: /^Game Mods folder vs / })
  await expect(dialog.getByText('Seed Stray')).toBeVisible()
  await dialog.getByRole('button', { name: 'Move into profile' }).click()
  await dialog.getByRole('button', { name: 'Close' }).click()
  await expect(page.getByRole('button', { name: 'Review…' })).toBeHidden()
})
