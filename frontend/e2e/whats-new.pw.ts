import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test("What's new opens from the Mortar menu after first show", async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('menuitem', { name: /^What is new in/ }).click()
  await expect(page.getByRole('dialog', { name: /^What is new in Mortar/ })).toBeVisible()
  await page.getByRole('button', { name: 'Close' }).click()
  await expect(page.getByRole('dialog', { name: /^What is new in Mortar/ })).toBeHidden()
})
