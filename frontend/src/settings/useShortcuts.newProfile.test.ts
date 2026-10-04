import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('Ctrl+N opens the naming dialog; duplicate reports errors', () => {
  const src = readFileSync(join(import.meta.dir, 'useShortcuts.ts'), 'utf8')
  expect(src).toContain('useCommandPalette.getState().setCreating(true)')
  expect(src).not.toContain("profiles.create('New profile')")
  expect(src).toContain('profiles.duplicate(profiles.openId).catch(reportUnexpected)')
  expect(src).not.toContain('.catch(() => undefined)')
})
