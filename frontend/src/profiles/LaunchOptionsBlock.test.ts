import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('launch presets confirm removal and same-name overwrite', () => {
  const src = readFileSync(join(import.meta.dir, 'LaunchOptionsBlock.tsx'), 'utf8')
  expect(src).toContain('<ConfirmDialog')
  expect(src).toContain('setRemoving(true)')
  expect(src).toContain('setOverwriting(true)')
  expect(src).toContain('t`Uses the saved options`')
})
