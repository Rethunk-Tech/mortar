import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('the Add split menu lists its sources', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'More ways to add mods' }).click()
  await expect(page.getByRole('menuitem')).toHaveText([
    'The downloads folder…',
    'Apply a template…',
    'Add a saved group…',
  ])
})
