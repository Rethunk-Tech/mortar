import { expect, test } from 'bun:test'
import { nexusModsUrl, nexusModUrl } from './nexusUrl.ts'

test('mod URLs default to the managed game and take a tab', () => {
  expect(nexusModsUrl()).toBe('https://www.nexusmods.com/stardewvalley/mods')
  expect(nexusModUrl(7)).toBe('https://www.nexusmods.com/stardewvalley/mods/7')
  expect(nexusModUrl('7', 'other', 'bugs')).toBe('https://www.nexusmods.com/other/mods/7?tab=bugs')
  expect(nexusModUrl(7, '')).toBe('https://www.nexusmods.com/stardewvalley/mods/7')
})
