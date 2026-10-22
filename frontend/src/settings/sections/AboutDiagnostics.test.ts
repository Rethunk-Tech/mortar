import { expect, test } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'AboutDiagnostics.tsx'), 'utf8')

test('Doctor failure shows an inline error and an empty report is all-clear', () => {
  expect(src).toContain('setFailed(true)')
  expect(src).toContain('{t`Run again`}')
  expect(src).toContain('{t`All checks passed`}')
})

test('every check code Go sends has a message', () => {
  const dir = join(import.meta.dir, '../../../../internal/doctor')
  const codes = new Set<string>()
  for (const file of readdirSync(dir).filter((f) => f.endsWith('.go') && !f.endsWith('_test.go'))) {
    for (const m of readFileSync(join(dir, file), 'utf8').matchAll(
      /\bCode(?::\s*|\s*=\s*)"(\w+)"/g,
    )) {
      codes.add(m[1] ?? '')
    }
  }
  const words = readFileSync(join(import.meta.dir, 'checkDetail.ts'), 'utf8')
  expect(codes.size).toBeGreaterThan(30)
  for (const code of codes) {
    expect(words).toContain(`case '${code}'`)
  }
})
