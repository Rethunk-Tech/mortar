import { expect, test } from 'bun:test'
import { hostOf, modActions } from './modActions.ts'

test('a mod with a page gets every action', () => {
  expect(
    modActions({ enabled: true, host: 'nexus', pinned: false, skipVersion: '', hasUpdate: false }),
  ).toEqual(['toggle', 'details', 'page', 'files', 'pin', 'remove'])
})

test('no page drops the page action', () => {
  expect(
    modActions({ enabled: false, host: '', pinned: false, skipVersion: '', hasUpdate: false }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'remove'])
})

test('an update can be skipped and a pin is always offered', () => {
  expect(
    modActions({ enabled: true, host: '', pinned: true, skipVersion: '', hasUpdate: true }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'skip', 'remove'])
  expect(
    modActions({ enabled: true, host: '', pinned: false, skipVersion: '2.0.0', hasUpdate: false }),
  ).toEqual(['toggle', 'details', 'files', 'pin', 'skip', 'remove'])
})

test('the page host follows the URL', () => {
  expect(hostOf(undefined)).toBe('')
  expect(hostOf('https://github.com/a/b')).toBe('github')
  expect(hostOf('https://www.nexusmods.com/stardewvalley/mods/1')).toBe('nexus')
  expect(hostOf('https://thunderstore.io/c/lethal-company/p/Ns/Mod/')).toBe('thunderstore')
  expect(hostOf('https://modrinth.com/mod/x')).toBe('web')
  expect(hostOf('https://notgithub.com/a/b')).toBe('web')
})
