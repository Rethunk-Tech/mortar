import { expect, test } from 'bun:test'
import { changesView, glance, renameSurface, savesView } from './homeView.ts'

test('changesView shows three lines until expanded', () => {
  const lines = ['a', 'b', 'c', 'd']
  expect(changesView(lines, false)).toEqual({ shown: ['a', 'b', 'c'], canExpand: true })
  expect(changesView(lines, true)).toEqual({ shown: lines, canExpand: false })
  expect(changesView(['a'], false).canExpand).toBe(false)
})

test('savesView drops the calendar for saves without one and counts the rest', () => {
  const base = { farm: 'Cookie', folder: 'Cookie_1', year: 3, season: 2, unrecorded: false }
  const lc = { farm: '', folder: 'LCSaveFile1', year: 0, season: 0, unrecorded: true }
  const view = savesView([base, lc, base, base, base, base])
  expect(view.shown).toHaveLength(4)
  expect(view.more).toBe(2)
  expect(view.shown[0]?.calendar).toEqual({ year: 3, season: 2 })
  expect(view.shown[1]?.calendar).toBeNull()
})

test('glance keeps an unread problem count null', () => {
  expect(glance(3, 0, null).problems).toBeNull()
})

test('a hidden hero sends a rename to the dialog', () => {
  expect(renameSurface('hidden')).toBe('dialog')
  expect(renameSurface('full')).toBe('inline')
  expect(renameSurface('compact')).toBe('inline')
})
