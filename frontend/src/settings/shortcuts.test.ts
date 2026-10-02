import { describe, expect, test } from 'bun:test'
import { matchShortcut, SHORTCUTS, shortcutAllowed } from './shortcuts.ts'

describe('SHORTCUTS', () => {
  test('lists every chord the handler recognises', () => {
    const ids = SHORTCUTS.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(ids).toContain('tab-performance')
    expect(ids).toContain('vanilla-play')
  })

  test('matchShortcut covers the table', () => {
    expect(matchShortcut({ key: 'k', ctrlKey: true })).toBe('command-palette')
    expect(matchShortcut({ key: 'f', ctrlKey: true })).toBe('filter-mods')
    expect(matchShortcut({ key: 'p', metaKey: true })).toBe('play')
    expect(matchShortcut({ key: 'F5' })).toBe('check-updates')
    expect(matchShortcut({ key: ',', ctrlKey: true })).toBe('open-settings')
    expect(matchShortcut({ key: 'Escape' })).toBe('dismiss')
    expect(matchShortcut({ key: 'a', ctrlKey: true })).toBe('select-all-mods')
    expect(matchShortcut({ key: 'ArrowUp' })).toBe('mod-up')
    expect(matchShortcut({ key: 'ArrowDown' })).toBe('mod-down')
    expect(matchShortcut({ key: ' ' })).toBe('mod-toggle')
    expect(matchShortcut({ key: 'Enter' })).toBe('mod-details')
    expect(matchShortcut({ key: 'Delete' })).toBe('mod-remove')
    expect(matchShortcut({ key: '6', ctrlKey: true })).toBe('tab-performance')
    expect(matchShortcut({ key: 'n', ctrlKey: true })).toBe('new-profile')
    expect(matchShortcut({ key: 'F2' })).toBe('rename-profile')
    expect(matchShortcut({ key: 'i', ctrlKey: true })).toBe('import')
    expect(matchShortcut({ key: 'ArrowLeft', altKey: true })).toBe('back')
  })
})

describe('shortcutAllowed', () => {
  test('blocks non-Esc while typing', () => {
    const input = { tagName: 'INPUT' }
    expect(shortcutAllowed('filter-mods', input)).toBe(false)
    expect(shortcutAllowed('dismiss', input)).toBe(true)
  })

  test('blocks non-Esc when a dialog is open', () => {
    expect(shortcutAllowed('play', { tagName: 'DIV' }, true)).toBe(false)
    expect(shortcutAllowed('dismiss', { tagName: 'DIV' }, true)).toBe(true)
  })

  test('allows chords on the page body', () => {
    expect(shortcutAllowed('check-updates', { tagName: 'DIV' })).toBe(true)
  })
})
