import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'PrefControls.tsx'), 'utf8')

test('PrefNumber keeps the draft and shows a min-max error instead of resetting', () => {
  expect(src).toContain('setInvalid(true)')
  expect(src).toContain('helperText={invalid ? range : undefined}')
  expect(src).not.toMatch(
    /if \(!Number\.isInteger\(n\) \|\| n < min \|\| n > max\) \{\s*setDraft\(String\(value\)\)/,
  )
})
