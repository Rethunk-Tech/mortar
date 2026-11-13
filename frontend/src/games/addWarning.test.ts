import { expect, test } from 'bun:test'
import { addGameWarning, storeAddWarning } from './addWarning.ts'

test('adding GOG Graveyard Keeper warns that plugins will not load, and does not warn for other stores', () => {
  const warning = addGameWarning('gog-32bit')
  expect(warning).toContain('32-bit')
  expect(warning).toContain('plugins will not load')
  expect(storeAddWarning('graveyard-keeper', 'gog')).toBe(warning)
  expect(storeAddWarning('graveyard-keeper', 'gog-heroic')).toBe(warning)
  expect(storeAddWarning('graveyard-keeper', 'gog-minigalaxy')).toBe(warning)
  expect(storeAddWarning('graveyard-keeper', 'steam')).toBe('')
  expect(storeAddWarning('stardew', 'gog')).toBe('')
  expect(addGameWarning('')).toBe('')
})
