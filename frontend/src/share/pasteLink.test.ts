import { expect, test } from 'bun:test'
import { classifyPastedLink, pasteTargetIsEditable } from './pasteLink.ts'

test('classifyPastedLink recognises share, collection and mod links', () => {
  expect(classifyPastedLink('https://mortar.rethunk.tech/stardew/p#abc')).toBe('share')
  expect(classifyPastedLink('  mortar://stardew/p/abc  ')).toBe('share')
  expect(classifyPastedLink('https://www.nexusmods.com/games/stardewvalley/collections/sve')).toBe(
    'collection',
  )
  expect(
    classifyPastedLink(
      'https://next.nexusmods.com/games/stardewvalley/collections/sve/revisions/3',
    ),
  ).toBe('collection')
  expect(classifyPastedLink('https://nexusmods.com/stardewvalley/mods/1915')).toBe('mod')
  expect(classifyPastedLink('https://www.nexusmods.com/games/stardewvalley/mods/1915')).toBe('mod')
})

test('classifyPastedLink rejects http, other hosts and plain text', () => {
  expect(classifyPastedLink('http://www.nexusmods.com/stardewvalley/mods/1915')).toBeNull()
  expect(classifyPastedLink('http://mortar.rethunk.tech/stardew/p#abc')).toBeNull()
  expect(classifyPastedLink('https://example.com/stardewvalley/mods/1915')).toBeNull()
  expect(classifyPastedLink('not a link')).toBeNull()
  expect(classifyPastedLink('')).toBeNull()
})

test('classifyPastedLink does not confuse collection, mod and share shapes', () => {
  expect(
    classifyPastedLink('https://www.nexusmods.com/games/stardewvalley/collections/sve'),
  ).not.toBe('mod')
  expect(classifyPastedLink('https://www.nexusmods.com/stardewvalley/mods/1915')).not.toBe(
    'collection',
  )
  expect(classifyPastedLink('https://mortar.rethunk.tech/stardew/p#abc')).not.toBe('mod')
  expect(classifyPastedLink('https://www.nexusmods.com/games/stardewvalley/mods/1915')).not.toBe(
    'share',
  )
})

test('pasteTargetIsEditable is true for fields and their descendants', () => {
  expect(pasteTargetIsEditable({ tagName: 'INPUT' })).toBe(true)
  expect(pasteTargetIsEditable({ tagName: 'TEXTAREA' })).toBe(true)
  expect(pasteTargetIsEditable({ tagName: 'SELECT' })).toBe(true)
  expect(pasteTargetIsEditable({ tagName: 'DIV', isContentEditable: true })).toBe(true)
  expect(
    pasteTargetIsEditable({
      tagName: 'SPAN',
      closest: () => ({}),
    }),
  ).toBe(true)
})

test('pasteTargetIsEditable is false for the page body and a plain div', () => {
  expect(pasteTargetIsEditable({ tagName: 'BODY' })).toBe(false)
  expect(pasteTargetIsEditable({ tagName: 'DIV' })).toBe(false)
  expect(pasteTargetIsEditable(null)).toBe(false)
})
