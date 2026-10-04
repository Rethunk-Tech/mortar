import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('the profiles page leaves on Escape only when no dialog is open and the target is not a field', () => {
  const src = readFileSync(join(import.meta.dir, 'ProfilesPage.tsx'), 'utf8')
  expect(src).toContain('shouldLeavePageOnEscape(e, dialogOpen())')
  expect(src).toContain('isTypingTarget(typing)')
})
