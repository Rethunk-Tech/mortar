import { describe, expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { formatChord, parseKeys } from '@rethunk/dom-input-nav'
import {
  boundShortcut,
  conflictFor,
  defaultBindings,
  matchShortcut,
  mergeBindings,
  SHORTCUTS,
  shortcutAllowed,
  takeableFrom,
} from './shortcuts.ts'

describe('SHORTCUTS', () => {
  test('ids are unique', () => {
    const ids = SHORTCUTS.map((s) => s.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  // Each modifier and each key whose table name differs from the browser's event key.
  test('matchShortcut reads modifiers and named keys as the browser sends them', () => {
    expect(matchShortcut({ key: 'k', ctrlKey: true })).toBe('command-palette')
    expect(matchShortcut({ key: 'p', metaKey: true })).toBe('play')
    expect(matchShortcut({ key: 'n', ctrlKey: true })).toBe('new-profile')
    expect(matchShortcut({ key: 'n', ctrlKey: true, shiftKey: true })).toBe('notifications')
    expect(matchShortcut({ key: 'ArrowLeft', altKey: true })).toBe('back')
    expect(matchShortcut({ key: 'ArrowLeft' })).toBeNull()
    expect(matchShortcut({ key: 'F5' })).toBe('check-updates')
    expect(matchShortcut({ key: 'Escape' })).toBe('dismiss')
    expect(matchShortcut({ key: 'ArrowUp' })).toBe('mod-up')
    expect(matchShortcut({ key: 'ArrowDown' })).toBe('mod-down')
    expect(matchShortcut({ key: ' ' })).toBe('mod-toggle')
  })
})

describe('combo parse and match', () => {
  test('parseKeys and formatChord round-trip the table', () => {
    for (const row of SHORTCUTS) {
      const parsed = parseKeys(row.keys)
      expect(parsed).not.toBeNull()
      expect(formatChord(parsed as NonNullable<typeof parsed>)).toBe(row.keys)
    }
  })

  test('matchShortcut uses live bindings', () => {
    const bindings = { ...defaultBindings(), play: 'Ctrl+Shift+L' }
    expect(matchShortcut({ key: 'p', ctrlKey: true }, bindings)).toBeNull()
    expect(matchShortcut({ key: 'l', ctrlKey: true, shiftKey: true }, bindings)).toBe('play')
  })

  test('conflictFor names the other action', () => {
    const bindings = mergeBindings({})
    expect(conflictFor('downloads', 'Ctrl+K', bindings)).toBe('command-palette')
    expect(conflictFor('downloads', 'Ctrl+J', bindings)).toBeNull()
    expect(conflictFor('command-palette', 'Ctrl+K', bindings)).toBeNull()
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

  test('lets chords through a checkbox', () => {
    expect(shortcutAllowed('filter-mods', { tagName: 'INPUT', type: 'checkbox' })).toBe(true)
  })

  test('allows chords on the page body', () => {
    expect(shortcutAllowed('check-updates', { tagName: 'DIV' })).toBe(true)
  })
})

test('a rebound list key triggers its action and the old key no longer does', () => {
  const bindings = { 'mod-toggle': 'T' }
  expect(boundShortcut({ key: 't' }, bindings)).toBe('mod-toggle')
  expect(boundShortcut({ key: ' ' }, bindings)).not.toBe('mod-toggle')
  expect(boundShortcut({ key: 'Enter' }, null)).toBe('mod-details')
})

test("the Go settings' default table is this one, so a rebind saves and no two tabs share a chord", () => {
  const go = readFileSync(
    new URL('../../../internal/settings/shortcuts.go', import.meta.url),
    'utf8',
  )
  const table = /var defaultShortcuts = map\[string\]string\{([^}]*)\}/.exec(go)?.[1] ?? ''
  const pairs = Object.fromEntries(
    [...table.matchAll(/"([^"]+)":\s*"([^"]+)"/g)].map((m) => [m[1], m[2]]),
  )
  expect(pairs).toEqual(defaultBindings())
})

test('an action stored unbound stays unbound and matches no key', () => {
  const bindings = mergeBindings({ play: 'Ctrl+K', 'command-palette': '' })
  expect(bindings['command-palette']).toBe('')
  expect(matchShortcut({ key: 'k', ctrlKey: true }, bindings)).toBe('play')
})

test("a default's chord can be taken from it, a chord the user chose cannot", () => {
  expect(takeableFrom('play', 'Ctrl+K', defaultBindings())).toBe('command-palette')
  const chosen = mergeBindings({ 'command-palette': 'Ctrl+J' })
  expect(takeableFrom('play', 'Ctrl+J', chosen)).toBeNull()
  expect(takeableFrom('play', 'Ctrl+Alt+F12', defaultBindings())).toBeNull()
})
