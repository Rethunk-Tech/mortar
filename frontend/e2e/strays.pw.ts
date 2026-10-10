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
  // Mortar refuses the move while a game process is still exiting, which the stand-in game of an earlier spec can be.
  const close = dialog.getByRole('button', { name: 'Close' })
  await expect(async () => {
    await dialog.getByRole('button', { name: 'Move into profile' }).click({ timeout: 1000 })
    await expect(close).toBeVisible({ timeout: 2000 })
  }).toPass({ timeout: 20_000 })
  await close.click()
  await expect(page.getByRole('button', { name: 'Review…' })).toBeHidden()
})
