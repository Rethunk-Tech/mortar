import { expect, test } from 'bun:test'
import { hostOf, modActions } from './modActions.ts'

test('a mod with a page gets every action', () => {
  expect(modActions({ enabled: true, host: 'nexus' })).toEqual([
    'toggle',
    'details',
    'page',
    'files',
    'remove',
  ])
})

test('no page drops the page action', () => {
  expect(modActions({ enabled: false, host: '' })).toEqual(['toggle', 'details', 'files', 'remove'])
})

test('the page host follows the URL', () => {
  expect(hostOf(undefined)).toBe('')
  expect(hostOf('https://github.com/a/b')).toBe('github')
  expect(hostOf('https://www.nexusmods.com/stardewvalley/mods/1')).toBe('nexus')
})
