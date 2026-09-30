import { expect, test } from 'bun:test'
import { hostOf, modActions } from './modActions.ts'

test('a mod with a page and a removable folder gets every action', () => {
  expect(modActions({ enabled: true, host: 'nexus', removable: true })).toEqual([
    'toggle',
    'details',
    'page',
    'files',
    'remove',
  ])
})

test('no page and not removable drops those actions', () => {
  expect(modActions({ enabled: false, host: '', removable: false })).toEqual([
    'toggle',
    'details',
    'files',
  ])
})

test('the page host follows the URL', () => {
  expect(hostOf(undefined)).toBe('')
  expect(hostOf('https://github.com/a/b')).toBe('github')
  expect(hostOf('https://www.nexusmods.com/stardewvalley/mods/1')).toBe('nexus')
})
