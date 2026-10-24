import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('every tidy code Go reports has a message', () => {
  const root = join(import.meta.dir, '../../..')
  const codes = new Set<string>()
  for (const file of ['main.go', 'internal/profile/ops.go']) {
    const go = readFileSync(join(root, file), 'utf8')
    for (const m of go.matchAll(/\b(?:tidied\.Add|s\.tidied)\("(\w+)"/g)) {
      codes.add(m[1] ?? '')
    }
  }
  const words = readFileSync(join(import.meta.dir, 'report.ts'), 'utf8')
  expect(codes.size).toBe(5)
  for (const code of codes) {
    expect(words).toContain(`case '${code}'`)
  }
})
