import { expect, test } from 'bun:test'
import type { PaletteItem } from './match.ts'
import { arrangePalette } from './sections.ts'

const items: PaletteItem[] = [
  { id: 'mod:k/a', kind: 'mod', label: 'Alpha' },
  { id: 'settings:general', kind: 'settings', label: 'General' },
  { id: 'action:play', kind: 'action', label: 'Play' },
  { id: 'tab:mods', kind: 'action', label: 'Switch to Mods' },
  { id: 'profile:p', kind: 'profile', label: 'Farm' },
  { id: 'toggle-mod:k/a', kind: 'action', label: 'Toggle Alpha' },
]

test('sections run Go to, Actions, Settings, Mods', () => {
  expect(arrangePalette(items, '', []).map((r) => [r.section, r.item.id])).toEqual([
    ['goto', 'profile:p'],
    ['goto', 'tab:mods'],
    ['actions', 'action:play'],
    ['settings', 'settings:general'],
    ['mods', 'mod:k/a'],
    ['mods', 'toggle-mod:k/a'],
  ])
})

test('recent picks lead once, only while nothing is typed', () => {
  const rows = arrangePalette(items, '', ['action:play', 'gone', 'settings:general'])
  expect(rows.slice(0, 2).map((r) => [r.section, r.item.id])).toEqual([
    ['recent', 'action:play'],
    ['recent', 'settings:general'],
  ])
  expect(rows.filter((r) => r.item.id === 'action:play')).toHaveLength(1)
  expect(arrangePalette(items, 'play', ['action:play'])[0]?.section).toBe('actions')
})

test('a long result list keeps the first fifty rows, cutting Mods first', () => {
  const many = Array.from({ length: 60 }, (_, i) => ({
    id: `mod:k/${i}`,
    kind: 'mod' as const,
    label: `Mod ${i}`,
  }))
  const rows = arrangePalette([...many, ...items], '', [])
  expect(rows).toHaveLength(50)
  expect(rows.some((r) => r.item.id === 'action:play')).toBe(true)
})
