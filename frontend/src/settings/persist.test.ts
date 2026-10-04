import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('persist reports failures with reportError, not raw Go text in body', () => {
  const src = readFileSync(join(import.meta.dir, 'persist.ts'), 'utf8')
  expect(src).toContain('run().catch(reportError(title))')
  expect(src).not.toContain('errorText')
})
