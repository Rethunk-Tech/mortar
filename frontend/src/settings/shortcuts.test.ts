import { describe, expect, test } from 'bun:test'
import { matchShortcut, SHORTCUTS, shortcutAllowed } from './shortcuts.ts'

describe('SHORTCUTS', () => {
  test('lists every chord the handler recognises', () => {
    const ids = SHORTCUTS.map((s) => s.id)
    expect(ids).toEqual(['filter-mods', 'play', 'check-updates', 'open-settings', 'dismiss'])
    expect(SHORTCUTS.map((s) => s.keys)).toEqual(['Ctrl+F', 'Ctrl+P', 'F5', 'Ctrl+,', 'Esc'])
  })

  test('matchShortcut covers the table', () => {
    expect(matchShortcut({ key: 'f', ctrlKey: true })).toBe('filter-mods')
    expect(matchShortcut({ key: 'p', metaKey: true })).toBe('play')
    expect(matchShortcut({ key: 'F5' })).toBe('check-updates')
    expect(matchShortcut({ key: ',', ctrlKey: true })).toBe('open-settings')
    expect(matchShortcut({ key: 'Escape' })).toBe('dismiss')
    expect(matchShortcut({ key: 'a' })).toBeNull()
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
