import { expect, type Page, test } from '@playwright/test'

const SETTLE_MS = 700

async function serverUp(baseURL: string): Promise<boolean> {
  try {
    return (await fetch(baseURL)).ok
  } catch {
    return false
  }
}

// Each step's spotlight must sit over the element the step talks about (in step order).
const STEP_TARGETS = [
  'main nav',
  '[data-tour="browse-tab"]',
  '[data-tour="game-tab"]',
  'main nav button.MuiButton-contained, main nav .MuiButtonGroup-root button',
  '[data-tour="mods-tab"]',
  'main [data-mod-row], main [data-mod-id]',
  'main [data-mod-row], main [data-mod-id]',
  '[data-tour="problems-tab"]',
  '[data-tour="app-menu"]',
]

function spotlightCovers(page: Page, selector: string): Promise<boolean> {
  return page.evaluate((sel) => {
    const spot = document.querySelector('[data-tour-spotlight]')?.getBoundingClientRect()
    const target = document.querySelector(sel)?.getBoundingClientRect()
    if (!(spot && target)) {
      return false
    }
    const x = target.left + target.width / 2
    const y = target.top + target.height / 2
    return x >= spot.left && x <= spot.right && y >= spot.top && y <= spot.bottom
  }, selector)
}

test('the replayed tour spotlights each step and Skip keeps it closed after a reload', async ({
  page,
  baseURL,
}) => {
  if (!(await serverUp(baseURL ?? ''))) {
    test
      .info()
      .annotations.push({ type: 'skipped', description: 'self-test server is not running' })
    return
  }
  await page.goto('/')
  await expect(page.getByRole('tab').first()).toBeVisible({ timeout: 30_000 })
  await page.getByRole('button', { name: 'Mortar menu' }).click()
  await page.getByRole('button', { name: 'Settings', exact: true }).click()
  await page.getByRole('button', { name: 'Show the tour again' }).click()
  await page.keyboard.press('Escape')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible({ timeout: 10_000 })

  for (const [i, selector] of STEP_TARGETS.entries()) {
    await page.waitForTimeout(SETTLE_MS)
    expect(await spotlightCovers(page, selector), `step ${i + 1} spotlight`).toBe(true)
    if (i < STEP_TARGETS.length - 1) {
      await dialog.getByRole('button', { name: 'Next' }).click()
    }
  }
  await dialog.getByRole('button', { name: 'Skip tour' }).click()
  await expect(dialog).toBeHidden()
  await page.reload()
  await expect(page.getByRole('tab').first()).toBeVisible({ timeout: 30_000 })
  await page.waitForTimeout(SETTLE_MS * 2)
  await expect(page.getByRole('dialog')).toBeHidden()
})
