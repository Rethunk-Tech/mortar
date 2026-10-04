import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('the strays callout moves the game-folder mod into the profile', async ({ page }) => {
  await openSeedFarm(page)
  await expect(page.getByText("1 mod in the game's Mods folder isn't in Mortar.")).toBeVisible()
  await page.getByRole('button', { name: 'Move into this profile…' }).click()
  const dialog = page.getByRole('dialog', { name: /^Move mods into / })
  await expect(dialog.getByText('Seed Stray')).toBeVisible()
  await dialog.getByRole('button', { name: 'Move 1 mod' }).click()
  await expect(page.getByText('Moved 1 mod')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Move into this profile…' })).toBeHidden()
})
