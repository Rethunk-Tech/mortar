import { expect, test } from '@playwright/test'
import { openGameSettings, openSeedFarm } from './app.ts'

test('Game settings › Mods trims a profile history to the changes kept', async ({ page }) => {
  await openSeedFarm(page)
  await openGameSettings(page)
  await page.getByRole('navigation').getByRole('button', { name: 'Mods', exact: true }).click()
  await page.getByRole('button', { name: 'Trim…' }).first().click()
  const dialog = page.getByRole('dialog', { name: /^Trim history of / })
  const keep = dialog.getByLabel('Changes to keep')
  await expect(keep).toBeVisible()
  await keep.fill('1')
  await dialog.getByRole('button', { name: /^Trim/ }).click()
  await expect(page.getByText(/^Trimmed .*'s history/)).toBeVisible()
})
