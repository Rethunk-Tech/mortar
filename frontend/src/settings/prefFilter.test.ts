import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { prefMatches } from './prefFilter.ts'

// The Lingui macro does not run under bun test, so labels are read from the source the catalog is extracted from.
function copyLabels(): Map<string, string> {
  const source = readFileSync(join(import.meta.dir, 'prefCopy.ts'), 'utf8')
  const labels = new Map<string, string>()
  for (const m of source.matchAll(/^ {4}([a-zA-Z0-9]+): \{\n\s+label: i18n\._\(msg`([^`]+)`\)/gm)) {
    labels.set(m[1] ?? '', m[2] ?? '')
  }
  return labels
}

test('every preference row is found by its own label', () => {
  const source = readFileSync(join(import.meta.dir, 'prefCopy.ts'), 'utf8')
  const keys = [...source.matchAll(/^ {4}([a-zA-Z0-9]+): \{$/gm)].map((m) => m[1] ?? '')
  const labels = copyLabels()
  expect(keys.length).toBeGreaterThan(50)
  for (const key of keys) {
    const label = labels.get(key)
    expect(label, `${key} has a label`).toBeTruthy()
    expect(prefMatches(label ?? '', label ?? '')).toBe(true)
    expect(prefMatches((label ?? '').toUpperCase(), label ?? '')).toBe(true)
  }
})

test('obvious synonyms reach the setting they mean', () => {
  expect(prefMatches('dark', 'Theme', '')).toBe(true)
  expect(prefMatches('light', 'Theme', '')).toBe(true)
  expect(prefMatches('proxy', 'LAN port', '')).toBe(true)
  expect(prefMatches('port', 'Automatic port', '')).toBe(true)
  expect(prefMatches('api key', 'Account', '')).toBe(true)
  expect(prefMatches('trash', 'Recently deleted retention', '')).toBe(true)
  expect(prefMatches('cleanup', 'Clean up…', '')).toBe(true)
  expect(prefMatches('dark', 'Language', '')).toBe(false)
})
