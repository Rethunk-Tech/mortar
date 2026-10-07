import { expect, type Page, test } from '@playwright/test'
import { openSeedFarm, openSettings } from './app.ts'

/** A task this long freezes the window noticeably; profile switches on a 400-mod profile stay under it. */
const LONG_TASK_MS = 200
const SETTLE_MS = 600
const HTTP_ERROR = 400

async function serverUp(baseURL: string): Promise<boolean> {
  try {
    return (await fetch(baseURL)).ok
  } catch {
    return false
  }
}

function watch(page: Page) {
  const errors: string[] = []
  page.on('console', (m) => {
    // A failed request is reported by the response handler with its URL; the console line repeats it without one.
    if (m.type() === 'error' && !m.text().startsWith('Failed to load resource')) {
      errors.push(m.text())
    }
  })
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('response', (r) => {
    if (r.status() >= HTTP_ERROR) {
      errors.push(`${r.status()} ${r.url()}`)
    }
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

test('every tab and settings page opens without errors or long tasks', async ({
  page,
  baseURL,
}) => {
  test.setTimeout(90_000)
  // Opt-in: without the self-test server (scripts/selftest.sh start) there is nothing to drive.
  if (!(await serverUp(baseURL ?? ''))) {
    test
      .info()
      .annotations.push({ type: 'skipped', description: 'self-test server is not running' })
    return
  }
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
