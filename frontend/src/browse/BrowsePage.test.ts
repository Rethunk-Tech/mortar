import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'BrowsePage.tsx'), 'utf8')

test('Nexus Download is only offered for Nexus results when the account is Premium', () => {
  expect(src).toContain('source === NEXUS && premium')
  expect(src).not.toMatch(/else if \(premium\)/)
})

test('free Nexus results show a disabled Download with a Premium-only reason', () => {
  expect(src).toContain('Premium only')
  expect(src).toContain('DisabledReason')
})
