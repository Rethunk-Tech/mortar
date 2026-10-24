import { expect, test } from 'bun:test'
import { hostOf, modActions } from './modActions.ts'

test('a mod with a page gets every action', () => {
  expect(
    modActions({
      enabled: true,
      host: 'nexus',
      pinned: false,
      skipVersion: '',
      hasUpdate: false,
      bundled: false,
      part: false,
      implicit: false,
    }),
  ).toEqual(['toggle', 'details', 'page', 'files', 'pin', 'reason', 'remove'])
})

test('no page drops the page action', () => {
  expect(
    modActions({
      enabled: false,
      host: '',
      pinned: false,
      skipVersion: '',
      hasUpdate: false,
      bundled: false,
      part: false,
      implicit: false,
    }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'reason', 'remove'])
})

test('an update can be skipped and a pin is always offered', () => {
  expect(
    modActions({
      enabled: true,
      host: '',
      pinned: true,
      skipVersion: '',
      hasUpdate: true,
      bundled: true,
      part: false,
      implicit: true,
    }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'skip', 'remove'])
  expect(
    modActions({
      enabled: true,
      host: '',
      pinned: false,
      skipVersion: '2.0.0',
      hasUpdate: false,
      bundled: false,
      part: false,
      implicit: false,
    }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'skip', 'reason', 'remove'])
})

test('the page host follows the URL', () => {
  expect(hostOf(undefined)).toBe('')
  expect(hostOf('https://github.com/a/b')).toBe('github')
  expect(hostOf('https://www.nexusmods.com/stardewvalley/mods/1')).toBe('nexus')
  expect(hostOf('https://thunderstore.io/c/lethal-company/p/Ns/Mod/')).toBe('thunderstore')
  expect(hostOf('https://modrinth.com/project/x')).toBe('modrinth')
  expect(hostOf('https://www.curseforge.com/projects/309243')).toBe('curseforge')
  expect(hostOf('https://x.itch.io/game')).toBe('itch')
  expect(hostOf('https://example.com/x')).toBe('web')
  expect(hostOf('https://notgithub.com/a/b')).toBe('web')
  expect(hostOf('https://www.patreon.com/posts/cool-mod-4242')).toBe('patreon')
  expect(hostOf('https://patreon.com.evil.test/posts/1')).toBe('web')
})
