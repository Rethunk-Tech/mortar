import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = readFileSync(join(import.meta.dir, 'SmapiVersionRow.tsx'), 'utf8')

test('latest SMAPI install uses the loader store toast, not a second local one', () => {
  expect(src).toContain('installLatest(GAME_STARDEW, selected === LATEST ? undefined : selected)')
  expect(src).not.toContain('InstallVersion')
  expect(src).not.toContain('is installed')
})
