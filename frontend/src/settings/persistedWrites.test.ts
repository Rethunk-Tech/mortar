import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative } from 'node:path'

const SRC = join(import.meta.dir, '..')

// Writes that mirror what the backend already holds (a read of the settings, or its change event), so they save nothing.
const MIRRORS = new Set(['share/arrivals.ts', 'settings/store.ts'])

function sources(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const path = join(dir, e.name)
    if (e.isDirectory()) {
      return e.name === 'locales' ? [] : sources(path)
    }
    return /\.tsx?$/.test(e.name) && !/\.test\./.test(e.name) ? [path] : []
  })
}

// A persisted key written to the settings store lives only until the app restarts unless the same function also calls
// its Set... binding, so every write sits within a few lines of one.
test('every write of a persisted setting to the store is followed by its save call', () => {
  const unsaved: string[] = []
  for (const file of sources(SRC)) {
    const rel = relative(SRC, file)
    if (MIRRORS.has(rel)) {
      continue
    }
    const lines = readFileSync(file, 'utf8').split('\n')
    for (const [i, line] of lines.entries()) {
      if (!line.includes('useSettings.setState(')) {
        continue
      }
      const after = lines.slice(i, i + 8).join('\n')
      if (!/\bSet[A-Z]\w*\(/.test(after)) {
        unsaved.push(`${rel}:${i + 1}`)
      }
    }
  }
  expect(unsaved).toEqual([])
})
