import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const read = (name: string) => readFileSync(join(import.meta.dir, name), 'utf8')
const hints = read('browseConstants.ts')
const actions = read('CardAction.tsx')

test('Nexus Download is only offered for Nexus results when the account is Premium', () => {
  expect(hints).toContain('source === NEXUS && premium')
  expect(actions).not.toMatch(/else if \(premium\)/)
})

test('free Nexus accounts get the files page and signed-out users a sign-in, never a dead Download', () => {
  expect(actions).toContain('Open files page')
  expect(actions).toContain('Sign in to download')
  expect(actions).not.toContain('Premium only')
})

test('the search function is one module-level value, so a parent re-render does not search again', () => {
  const host = read('BrowseHost.tsx')
  expect(host).toMatch(/^const search: BrowseSearch = /m)
  expect(host).not.toMatch(/^\s+const search\b/m)
})
