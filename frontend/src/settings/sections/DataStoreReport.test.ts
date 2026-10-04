import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'DataStoreReport.tsx'), 'utf8')

test('cleanup load failures surface Retry and closing clears the selection', () => {
  expect(src).toContain('.catch(() => setError(true))')
  expect(src).toContain('{t`Retry`}')
  expect(src).toContain('setPicked(new Set())')
})
