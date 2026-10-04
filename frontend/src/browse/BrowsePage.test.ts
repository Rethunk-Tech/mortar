import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'BrowsePage.tsx'), 'utf8')

test('Nexus Download is only offered for Nexus results when the account is Premium', () => {
  expect(src).toContain('source === NEXUS && premium')
  expect(src).not.toMatch(/else if \(premium\)/)
})

test('free Nexus accounts get the files page and signed-out users a sign-in, never a dead Download', () => {
  expect(src).toContain('Open files page')
  expect(src).toContain('Sign in to download')
  expect(src).not.toContain('Premium only')
})
