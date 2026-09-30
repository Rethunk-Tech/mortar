import { expect, test } from 'bun:test'
import { joinSummary, knownCount, originLine } from './summary.ts'

const labels = {
  link: 'imported from a link',
  mortar: 'imported from a .mortar file',
  gameMods: "imported from the game's Mods folder",
  copy: (name: string) => `copy of ${name}`,
}

test('origin is omitted when unknown', () => {
  expect(originLine(undefined, undefined, labels)).toBeUndefined()
  expect(originLine('new', undefined, labels)).toBeUndefined()
  expect(originLine('copy', undefined, labels)).toBeUndefined()
})

test('origin matches the path that created the profile', () => {
  expect(originLine('link', undefined, labels)).toBe('imported from a link')
  expect(originLine('mortar', undefined, labels)).toBe('imported from a .mortar file')
  expect(originLine('game-mods', undefined, labels)).toBe("imported from the game's Mods folder")
  expect(originLine('copy', 'Farm', labels)).toBe('copy of Farm')
})

test('cached counts are omitted until known and positive', () => {
  expect(knownCount(undefined, '3 updates')).toBeUndefined()
  expect(knownCount(0, '0 updates')).toBeUndefined()
  expect(knownCount(3, '3 updates')).toBe('3 updates')
})

test('the row summary joins the parts the mock shows', () => {
  expect(
    joinSummary([
      '42 mods',
      knownCount(3, '3 updates'),
      knownCount(1, '1 problem'),
      originLine('link', undefined, labels),
    ]),
  ).toBe('42 mods · 3 updates · 1 problem · imported from a link')
  expect(joinSummary(['2 mods', knownCount(undefined, '1 update')])).toBe('2 mods')
})
