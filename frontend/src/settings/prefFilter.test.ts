import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { prefMatches } from './prefFilter.ts'

// Every module prefCopy() consults, so a row's label is asserted wherever its copy lives.
const copySources = () =>
  readdirSync(import.meta.dir)
    .filter(
      (file) => file.startsWith('prefCopy') && file.endsWith('.ts') && !file.endsWith('.test.ts'),
    )
    .concat('vrPrefCopy.ts')
    .map((file) => readFileSync(join(import.meta.dir, file), 'utf8'))

// The Lingui macro does not run under bun test, so labels are read from the source the catalog is extracted from.
function copyLabels(): Map<string, string> {
  const labels = new Map<string, string>()
  for (const source of copySources()) {
    for (const m of source.matchAll(
      /^ {4}([a-zA-Z0-9]+): \{\n\s+label: i18n\._\(msg`([^`]+)`\)/gm,
    )) {
      labels.set(m[1] ?? '', m[2] ?? '')
    }
  }
  return labels
}

test('every preference row is found by its own label', () => {
  const keys = copySources().flatMap((source) =>
    [...source.matchAll(/^ {4}([a-zA-Z0-9]+): \{$/gm)].map((m) => m[1] ?? ''),
  )
  const labels = copyLabels()
  expect(keys.length).toBeGreaterThan(50)
  for (const key of keys) {
    const label = labels.get(key)
    expect(label, `${key} has a label`).toBeTruthy()
    expect(prefMatches(label ?? '', label ?? '')).toBe(true)
    expect(prefMatches((label ?? '').toUpperCase(), label ?? '')).toBe(true)
  }
})

test('the copy modules the filter consults are covered', () => {
  const labels = copyLabels()
  for (const key of [
    'antivirus',
    'graphics',
    'defaultModsView',
    'enableRequirements',
    'startFirst',
  ]) {
    expect(labels.get(key), `${key} has a label`).toBeTruthy()
  }
})

test('a query matches a label, a description or a synonym, and an empty one matches all', () => {
  expect(prefMatches('', 'Backup before Play', 'When Mortar zips Saves')).toBe(true)
  expect(prefMatches('zips', 'Backup before Play', 'When Mortar zips Saves')).toBe(true)
  expect(prefMatches('dark', 'Theme', '')).toBe(true)
  expect(prefMatches('prox', 'LAN port', '')).toBe(true)
  expect(prefMatches('pr', 'LAN port', '')).toBe(false)
  expect(prefMatches('light', 'Theme', '')).toBe(true)
  expect(prefMatches('api key', 'Account', '')).toBe(true)
  expect(prefMatches('trash', 'Recently deleted retention', '')).toBe(true)
  expect(prefMatches('cleanup', 'Clean up…', '')).toBe(true)
  expect(prefMatches('health', 'Connection', '')).toBe(true)
  expect(prefMatches('rate limit', 'Requests', '')).toBe(true)
  expect(prefMatches('dark', 'Language', '')).toBe(false)
})
