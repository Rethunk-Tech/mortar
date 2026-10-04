import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('game Mods folder preview errors offer Retry that reloads the preview', () => {
  const src = readFileSync(join(import.meta.dir, 'GameModsDialog.tsx'), 'utf8')
  expect(src).toContain('const loadPreview = useCallback')
  expect(src).toContain('onClick={loadPreview}')
  expect(src).toContain('t`Retry`')
})
