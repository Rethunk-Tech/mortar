import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('every console line code Go sends has a message', () => {
  const go = readFileSync(join(import.meta.dir, '../../../internal/launchsvc/service.go'), 'utf8')
  const codes = new Set<string>()
  for (const m of go.matchAll(
    /sayCode\([\s\S]*?, "(\w+)",\s*(?:nil|map\[|params)|\bcode = "(\w+)"/g,
  )) {
    codes.add(m[1] ?? m[2] ?? '')
  }
  const words = readFileSync(join(import.meta.dir, 'lineText.ts'), 'utf8')
  expect(codes.size).toBeGreaterThan(6)
  for (const code of codes) {
    expect(words).toContain(`case '${code}'`)
  }
})
