import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('the Problems tab lists the missing requirement of each seeded mod', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('tab', { name: /^Problems/ }).click()
  for (const mod of ['Seed Alpha', 'Seed Beta', 'Seed Gamma']) {
    await expect(
      page.getByRole('button', {
        name: `${mod} needs Pathoschild.ContentPatcher, which this profile lacks.`,
      }),
    ).toBeVisible()
  }
})
