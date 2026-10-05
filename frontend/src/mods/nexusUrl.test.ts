import { expect, test } from 'bun:test'
import { nexusModsUrl, nexusModUrl } from './nexusUrl.ts'

test('mod URLs take the game domain and a tab', () => {
  expect(nexusModsUrl('stardewvalley')).toBe('https://www.nexusmods.com/stardewvalley/mods')
  expect(nexusModUrl(7, 'stardewvalley')).toBe('https://www.nexusmods.com/stardewvalley/mods/7')
  expect(nexusModUrl('7', 'other', 'bugs')).toBe('https://www.nexusmods.com/other/mods/7?tab=bugs')
})
