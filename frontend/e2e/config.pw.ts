import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import process from 'node:process'
import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm } from './app.ts'
import { serverEnv } from './sandbox.ts'

const dir = process.env.MORTAR_E2E_DIR ?? ''

/** Seed Alpha's config.json in the Seed Farm profile, parsed from disk, so a bool can be told from a string. */
const alphaConfig = (): Record<string, unknown> => {
  const home = serverEnv(dir).HOME ?? ''
  const [found] = execFileSync(
    'find',
    [`${home}/.local/share/mortar`, '-path', '*/profiles/stardew/*/Seed.Alpha/config.json'],
    { encoding: 'utf8' },
  )
    .trim()
    .split('\n')
  return JSON.parse(readFileSync(found ?? '', 'utf8'))
}

async function openConfig(page: Page) {
  await page.getByRole('tab', { name: 'Config' }).click()
  await expect(page.getByRole('group', { name: 'Mods with settings' })).toBeVisible()
}

const modRow = (page: Page, name: string) =>
  page.getByRole('group', { name: 'Mods with settings' }).getByRole('button', { name })

test('the Config tab lists the seeded mods with their chips', async ({ page }) => {
  await openSeedFarm(page)
  await openConfig(page)
  await expect(modRow(page, /Seed Alpha/)).toContainText('Changed')
  await expect(modRow(page, /Seed Beta/)).toContainText(/1 waiting|In-game menu/)
})

test('Seed Alpha shows typed widgets, and the switch writes a JSON bool', async ({ page }) => {
  await openSeedFarm(page)
  await openConfig(page)
  await modRow(page, /Seed Alpha/).click()
  const pane = page.getByRole('region', { name: 'Config of Seed Alpha' })
  await expect(pane.getByRole('spinbutton', { name: 'Speed' })).toHaveValue('5')
  const enabled = pane.getByRole('switch', { name: 'Enabled' })
  await expect(enabled).toBeChecked()
  await enabled.click()
  await expect(enabled).not.toBeChecked()
  await expect.poll(() => alphaConfig().Enabled).toBe(false)
  await enabled.click()
  await expect.poll(() => alphaConfig().Enabled).toBe(true)
})

test('Seed Beta renders its in-game menu capture and the page stays responsive', async ({
  page,
}) => {
  await openSeedFarm(page)
  await openConfig(page)
  await modRow(page, /Seed Beta/).click()
  const pane = page.getByRole('region', { name: 'Config of Seed Beta' })
  await expect(pane.getByRole('switch', { name: 'Enabled' })).toBeVisible()
  await expect(pane.getByText('Count', { exact: true })).toBeVisible()
  // A render loop would starve the compositor while script still ran; both round trips must finish.
  await page.waitForTimeout(2000)
  expect(await page.evaluate(() => 1 + 1)).toBe(2)
  const shot = await page.screenshot({ timeout: 5000 })
  expect(shot.length).toBeGreaterThan(0)
})

test('Edit config in a mod menu opens the Config tab on that mod', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByText('Seed Alpha', { exact: true }).first().click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Edit config' }).click()
  await expect(page.getByRole('tab', { name: 'Config' })).toHaveAttribute('aria-selected', 'true')
  await expect(page.getByRole('region', { name: 'Config of Seed Alpha' })).toBeVisible()
  await expect(modRow(page, /Seed Alpha/)).toHaveClass(/Mui-selected/)
})

test('Lethal Company lists SeedAlpha and BepInEx.cfg', async ({ page }) => {
  await openSeedFarm(page)
  await page.getByRole('button', { name: 'Stardew Valley' }).first().click()
  await page.getByRole('menuitemradio', { name: 'Lethal Company' }).click()
  await expect(page.getByRole('button', { name: 'Seed Lobby' }).first()).toBeVisible()
  await openConfig(page)
  await modRow(page, /^SeedAlpha/).click()
  const pane = page.getByRole('region', { name: 'Config of SeedAlpha' })
  await expect(pane.getByRole('switch', { name: 'Enabled' })).not.toBeChecked()
  await expect(page.getByText('Loader and other')).toBeVisible()
  await expect(modRow(page, /BepInEx\.cfg/)).toBeVisible()
})
