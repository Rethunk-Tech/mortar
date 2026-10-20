// Checks a built website demo (bun scripts/site-demo.ts <dir>) served under /demo/: bun frontend/e2e/demo-check.ts <dir>
import { join, normalize } from 'node:path'
import process from 'node:process'
import { chromium, expect } from '@playwright/test'

const [, , dir] = process.argv
if (!dir) {
  console.error('usage: bun frontend/e2e/demo-check.ts <built demo dir>')
  process.exit(2)
}
const PREFIX = '/demo/'

const server = Bun.serve({
  hostname: '127.0.0.1',
  port: 0,
  fetch(request) {
    const path = new URL(request.url).pathname
    if (!path.startsWith(PREFIX)) {
      return new Response('outside the demo', { status: 404 })
    }
    const file = Bun.file(join(dir, normalize(path.slice(PREFIX.length) || 'index.html')))
    return file
      .exists()
      .then((found) => new Response(found ? file : 'missing', { status: found ? 200 : 404 }))
  },
})
const origin = `http://127.0.0.1:${server.port}`

const problems: string[] = []
const browser = await chromium.launch()
try {
  const page = await browser.newPage({
    viewport: { width: 1280, height: 800 },
    colorScheme: 'dark',
  })
  page.on('console', (m) => m.type() === 'error' && problems.push(`console: ${m.text()}`))
  page.on('pageerror', (e) => problems.push(`page error: ${e.message}`))
  page.on('request', (r) => {
    if (new URL(r.url()).origin !== origin && !r.url().startsWith('data:')) {
      problems.push(`left the origin: ${r.url()}`)
    }
  })
  page.on('response', (r) => r.status() >= 400 && problems.push(`${r.status()}: ${r.url()}`))

  await page.goto(`${origin}${PREFIX}`)
  await expect(page.getByRole('tab', { name: 'Mods' })).toHaveAttribute('aria-selected', 'true', {
    timeout: 15_000,
  })
  const details = page.getByRole('button', { name: /^Details of / })
  await expect(details).toHaveCount(3)
  await details.first().click()
  await expect(page.getByRole('button', { name: /^Pin (this )?version$/ }).first()).toBeVisible()

  await page.getByRole('tab', { name: 'Problems' }).click()
  await page.getByRole('button', { name: 'Why?' }).first().click()
  await expect(
    page.getByText('Seed Gamma needs Content Patcher, which this profile lacks.'),
  ).toBeVisible()
} finally {
  await browser.close()
  await server.stop()
}
if (problems.length > 0) {
  console.error(problems.join('\n'))
  process.exit(1)
}
console.log('demo ok: Mods list, a mod’s details and Problems render, offline, with no errors')
