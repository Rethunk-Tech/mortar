import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('first-run launcher load uses LoadingRow and a Retry panel on failure', () => {
  const src = readFileSync(join(import.meta.dir, 'FirstRun.tsx'), 'utf8')
  expect(src).not.toContain('return null')
  expect(src).toContain('<LoadingRow>{t`Loading…`}</LoadingRow>')
  expect(src).toContain('<LoadErrorRow')
  expect(src).toContain('setLoadError(inlineError(e))')
})
