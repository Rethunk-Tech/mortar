import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'DataUsageLoad.ts'), 'utf8')

test('a Usage failure sets error so the row can Retry via restart', () => {
  expect(src).toContain('setError(true)')
  expect(src).toContain('return { usage, bytes, restart, error }')
})
