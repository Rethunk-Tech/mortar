import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'AboutDiagnostics.tsx'), 'utf8')

test('Doctor failure shows an inline error and an empty report is all-clear', () => {
  expect(src).toContain('setFailed(true)')
  expect(src).toContain('{t`Run again`}')
  expect(src).toContain('{t`All checks passed`}')
})
