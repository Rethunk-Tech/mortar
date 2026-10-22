import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const PATTERNS = [
  /\bCode:\s*"(\w+)"/g,
  /code, params := "(\w+)"/g,
  /\.Code, \w+\.Params = "(\w+)"/g,
  /reason, code, params :?= "[^"]*"[^,]*, "(\w+)"/g,
  /failedCode\([^\n]*"(\w+)", nil\)/g,
  /hint\.Code = "(\w+)"/g,
  /ModKey:[^\n]*Reason:\s*"([\w-]+)"/g,
  /hintWhy\{[^\n]*, "(\w+)", map/g,
]

test('every reason code Go sends has a message', () => {
  const root = join(import.meta.dir, '../../../internal')
  const codes = new Set<string>()
  for (const file of [
    'loader/bepinex5/analyze.go',
    'loader/thunderpack/logs.go',
    'problems/loader.go',
    'problems/plugindeps.go',
    'avscan/product.go',
    'problems/cleanup.go',
    'framework/contentpatcher/settings_recolour.go',
    'framework/contentpatcher/tilesheets.go',
    'profile/gamemods.go',
    'profile/import.go',
    'launchsvc/runs.go',
    'launchsvc/crashhint.go',
    'launchsvc/dumphint.go',
  ]) {
    const go = readFileSync(join(root, file), 'utf8')
    for (const re of PATTERNS) {
      for (const m of go.matchAll(re)) {
        codes.add(m[1] ?? '')
      }
    }
  }
  const words = readFileSync(join(import.meta.dir, 'reasonText.ts'), 'utf8')
  expect(codes.size).toBeGreaterThan(12)
  for (const code of codes) {
    expect(words).toContain(`case '${code}'`)
  }
})
