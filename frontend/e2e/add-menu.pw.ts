import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('the Add split menu lists its three sources and the extra folder dialog adds 2 mods', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'More ways to add mods' }).click()
  await expect(page.getByRole('menuitem')).toHaveText([
    'Archive…',
    'From the downloads folder…',
    'From the extra mods folder…',
  ])
  await page.getByRole('menuitem', { name: 'From the extra mods folder…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Add from the extra mods folder' })
  await expect(dialog.getByText('Seed Extra One')).toBeVisible()
  await expect(dialog.getByText('Seed Extra Two')).toBeVisible()
  await dialog.getByRole('button', { name: 'Add 2 mods' }).click()
  await expect(dialog).toBeHidden()
  await page.getByRole('tab', { name: 'Mods' }).click()
  await expect(page.getByText('Seed Extra One', { exact: true })).toBeVisible()
  await expect(page.getByText('Seed Extra Two', { exact: true })).toBeVisible()
})
