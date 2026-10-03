import { expect, test } from 'bun:test'
import { nexusAuthorPageUrl } from './nexusAuthorPage.ts'

test('prefers Nexus user URL when member id is present', () => {
  expect(
    nexusAuthorPageUrl('https://www.nexusmods.com/users/1552317', 'https://example.com/mod'),
  ).toBe('https://www.nexusmods.com/users/1552317')
})

test('falls back to mod page when uploader URL is missing', () => {
  expect(nexusAuthorPageUrl('', 'https://www.nexusmods.com/stardewvalley/mods/1')).toBe(
    'https://www.nexusmods.com/stardewvalley/mods/1',
  )
})
