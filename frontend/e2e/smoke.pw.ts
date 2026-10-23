import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm, openSettings } from './app.ts'
import { methodOf } from './renderMeter.ts'

/** A task this long freezes the window noticeably; profile switches on a 400-mod profile stay under it. */
const LONG_TASK_MS = 200
const SETTLE_MS = 600
const HTTP_ERROR = 400
const NEXUS_SIGNED_OUT = 'sign in to Nexus Mods'

function watch(page: Page) {
  const errors: string[] = []
  page.on('console', (m) => {
    // A failed request is reported by the response handler with its URL; the console line repeats it without one.
    if (m.type() === 'error' && !m.text().startsWith('Failed to load resource')) {
      errors.push(m.text())
    }
  })
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('response', async (r) => {
    if (r.status() < HTTP_ERROR) {
      return
    }
    // The sandbox has no Nexus account, so Browse says why Nexus has no categories, as it does for a player who has not
    // signed in; any other failure is a fault.
    if ((await r.text().catch(() => '')).includes(NEXUS_SIGNED_OUT)) {
      return
    }
    // A backend call is named by its bound method, which the request body carries.
    const path = new URL(r.url()).pathname
    const what = path.startsWith('/wails/') ? methodOf(r.request().postData() ?? '') : r.url()
    errors.push(`${r.status()} ${what}`)
  })
  return errors
}

function longTasks(page: Page): Promise<number[]> {
  return page.evaluate(() =>
    (globalThis as unknown as { mortarLongTasks: number[] }).mortarLongTasks.splice(0),
  )
}

async function settle(page: Page) {
  await page.waitForTimeout(SETTLE_MS)
}

test('every tab and settings page opens without errors or long tasks', async ({ page }) => {
  test.setTimeout(90_000)
  await page.addInitScript(() => {
    const w = globalThis as unknown as { mortarLongTasks: number[] }
    w.mortarLongTasks = []
    new PerformanceObserver((list) => {
      for (const e of list.getEntries()) {
        w.mortarLongTasks.push(Math.round(e.duration))
      }
    }).observe({ type: 'longtask', buffered: false })
  })
  const errors = watch(page)
  await openSeedFarm(page)
  await settle(page)
  await longTasks(page)

  const slow: string[] = []
  const tabs = page.getByRole('tab')
  for (let i = 0; i < (await tabs.count()); i++) {
    const name = (await tabs.nth(i).textContent()) ?? `tab ${i}`
    await tabs.nth(i).click()
    await settle(page)
    const long = (await longTasks(page)).filter((ms) => ms > LONG_TASK_MS)
    if (long.length > 0) {
      slow.push(`${name}: ${long.join(', ')} ms`)
    }
  }

  await openSettings(page)
  const pages = page.getByRole('navigation').getByRole('button')
  for (let i = 0; i < (await pages.count()); i++) {
    const label = (await pages.nth(i).textContent()) ?? `page ${i}`
    await pages.nth(i).click()
    await settle(page)
    const long = (await longTasks(page)).filter((ms) => ms > LONG_TASK_MS)
    if (long.length > 0) {
      slow.push(`Settings › ${label}: ${long.join(', ')} ms`)
    }
  }

  expect(errors, 'console errors').toEqual([])
  expect(slow, 'long tasks').toEqual([])
})
