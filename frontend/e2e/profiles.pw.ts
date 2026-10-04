import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('New profile starts empty by default and can start from Seed Template', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'New profile' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'New profile' })
  await expect(dialog.getByLabel('Start from')).toHaveText('Empty profile')
  await dialog.getByLabel('Start from').click()
  await page.getByRole('option', { name: 'Seed Template' }).click()
  await dialog.getByLabel('Profile name').fill('From E2E')
  await dialog.getByRole('button', { name: 'Create' }).click()
  await expect(page.getByText(/^Created From E2E/)).toBeVisible()
  await expect(page.getByRole('button', { name: /^From E2E/ })).toBeVisible()
})

test('the profile menu saves a profile as a template', async ({ page }) => {
  await openSeedFarm(page)
  await page
    .getByRole('button', { name: /^Seed Farm/ })
    .first()
    .click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Save as template…' }).click()
  const dialog = page.getByRole('dialog', { name: 'Save as template' })
  await dialog.getByLabel('Template name').fill('E2E Template')
  await dialog.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('Template saved')).toBeVisible()
})
