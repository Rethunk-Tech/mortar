import { expect, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'

test('Light theme paints the app light and shows a decodable background preview', async ({
  page,
}) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('button', { name: 'Settings', exact: true }).click()
  await page
    .getByRole('navigation', { name: 'Settings sections' })
    .getByRole('button', { name: 'Appearance' })
    .click()
  await page.getByRole('button', { name: 'Light', exact: true }).click()
  const luminance = (css: string) => {
    const [r = 0, g = 0, b = 0] = (css.match(/\d+/g) ?? []).map(Number)
    return (r + g + b) / 3
  }
  await expect
    .poll(() =>
      page.evaluate(() => getComputedStyle(document.body).backgroundColor).then(luminance),
    )
    .toBeGreaterThan(200)
  expect(luminance(await page.evaluate(() => getComputedStyle(document.body).color))).toBeLessThan(
    100,
  )
  const preview = page.getByRole('img', { name: 'Background preview' })
  await expect(preview).toBeVisible()
  await expect
    .poll(() => preview.evaluate((img: HTMLImageElement) => img.naturalWidth))
    .toBeGreaterThan(0)
})
