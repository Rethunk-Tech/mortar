import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('Downloads history offers Retry all failed for the seeded failed download', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.keyboard.press('Control+j')
  await page
    .getByRole('dialog', { name: 'Downloads' })
    .getByRole('button', { name: 'History' })
    .click()
  await expect(page.getByText('Seed Failed Download')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Retry all failed' })).toBeVisible()
})
