import { expect, test } from 'bun:test'
import { sidebarGroups, tabUnavailable } from './sidebarTabs.ts'

const full = { order: true, console: true, startup: true }
const none = { order: false, console: false, startup: false }

test('the sidebar lists every group for a loader with all capabilities, in order', () => {
  const groups = sidebarGroups(full, { updates: 0, problems: 0 })
  expect(groups.map((g) => [g.id, g.entries.map((e) => e.id)])).toEqual([
    ['library', ['mods', 'saves', 'config']],
    ['get', ['browse']],
    ['health', ['problems', 'load-order', 'performance']],
    ['logs', ['console']],
  ])
})

test('sections a loader lacks are left out and an empty group loses its header', () => {
  const groups = sidebarGroups(none, { updates: 0, problems: 0 })
  expect(groups.map((g) => g.id)).toEqual(['library', 'get', 'health'])
  expect(groups[2]?.entries.map((e) => e.id)).toEqual(['problems'])
  expect(sidebarGroups({ ...none, console: true }, { updates: 0, problems: 0 }).at(-1)?.id).toBe(
    'logs',
  )
})

test('Mods carries the update count and Problems the problem count, each only when above zero', () => {
  const badges = (updates: number, problems: number | null) =>
    Object.fromEntries(
      sidebarGroups(full, { updates, problems })
        .flatMap((g) => g.entries)
        .map((e) => [e.id, e.badge]),
    )
  expect(badges(4, 3)).toMatchObject({
    mods: { n: 4, tone: 'primary' },
    problems: { n: 3, tone: 'warning' },
    saves: null,
  })
  expect(badges(0, null)).toMatchObject({ mods: null, problems: null })
  expect(badges(0, 0)).toMatchObject({ problems: null })
})

test('Config is in the Library group for a loader without any capability', () => {
  const library = sidebarGroups(none, { updates: 0, problems: 0 }).find((g) => g.id === 'library')
  expect(library?.entries.map((e) => e.id)).toEqual(['mods', 'saves', 'config'])
  expect(tabUnavailable('config', none)).toBe(false)
})

test('a saved section the loader lacks is unavailable', () => {
  expect(tabUnavailable('load-order', none)).toBe(true)
  expect(tabUnavailable('console', none)).toBe(true)
  expect(tabUnavailable('performance', none)).toBe(true)
  expect(tabUnavailable('mods', none)).toBe(false)
  expect(tabUnavailable('console', full)).toBe(false)
})
