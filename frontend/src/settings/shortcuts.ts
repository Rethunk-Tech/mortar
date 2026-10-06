const toEventKey: Record<string, string> = {
  Esc: 'Escape',
  Escape: 'Escape',
  '↑': 'ArrowUp',
  ArrowUp: 'ArrowUp',
  '↓': 'ArrowDown',
  ArrowDown: 'ArrowDown',
  Left: 'ArrowLeft',
  ArrowLeft: 'ArrowLeft',
  Right: 'ArrowRight',
  ArrowRight: 'ArrowRight',
  Space: ' ',
  ' ': ' ',
}

const fromEventKey: Record<string, string> = {
  Escape: 'Esc',
  ArrowUp: '↑',
  ArrowDown: '↓',
  ArrowLeft: 'Left',
  ArrowRight: 'Right',
  ' ': 'Space',
}

const modifiers = new Set(['Control', 'Shift', 'Alt', 'Meta', 'OS', 'Hyper', 'Super'])
let capturing = false

function eventKey(key: string): string {
  const mapped = toEventKey[key]
  if (mapped) {
    return mapped
  }
  if (key.length === 1) {
    return key.toLowerCase()
  }
  return key
}

function chordKey(c: {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
}): string {
  const ctrl = Boolean(c.ctrlKey || c.metaKey)
  return `${ctrl ? 1 : 0}${c.shiftKey ? 1 : 0}${c.altKey ? 1 : 0}${eventKey(c.key)}`
}

export type ShortcutId =
  | 'command-palette'
  | 'filter-mods'
  | 'play'
  | 'check-updates'
  | 'open-settings'
  | 'dismiss'
  | 'select-all-mods'
  | 'mod-up'
  | 'mod-down'
  | 'mod-toggle'
  | 'mod-details'
  | 'mod-remove'
  | 'tab-browse'
  | 'tab-mods'
  | 'tab-problems'
  | 'tab-load-order'
  | 'tab-saves'
  | 'tab-notes'
  | 'tab-console'
  | 'tab-performance'
  | 'new-profile'
  | 'duplicate-profile'
  | 'rename-profile'
  | 'find-all-mods'
  | 'import'
  | 'export-profile'
  | 'downloads'
  | 'notifications'
  | 'previous-profile'
  | 'next-profile'
  | 'collapse-sidebar'
  | 'back'
  | 'help'
  | 'vanilla-play'

export interface Shortcut {
  id: ShortcutId
  keys: string
  always: boolean
  group: 'General' | 'Navigation' | 'Profiles' | 'Mods list' | 'Tabs'
}

export interface Chord {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
}

export interface TypingTarget {
  tagName?: string
  isContentEditable?: boolean
}

export type ShortcutBindings = Partial<Record<ShortcutId, string>>

/** Single table for key handling and Settings › Shortcuts. */
export const SHORTCUTS: readonly Shortcut[] = [
  { id: 'command-palette', keys: 'Ctrl+K', always: false, group: 'General' },
  { id: 'filter-mods', keys: 'Ctrl+F', always: false, group: 'Mods list' },
  { id: 'play', keys: 'Ctrl+P', always: false, group: 'General' },
  { id: 'check-updates', keys: 'F5', always: false, group: 'General' },
  { id: 'open-settings', keys: 'Ctrl+,', always: false, group: 'General' },
  { id: 'dismiss', keys: 'Esc', always: true, group: 'General' },
  { id: 'select-all-mods', keys: 'Ctrl+A', always: false, group: 'Mods list' },
  { id: 'mod-up', keys: '↑', always: false, group: 'Mods list' },
  { id: 'mod-down', keys: '↓', always: false, group: 'Mods list' },
  { id: 'mod-toggle', keys: 'Space', always: false, group: 'Mods list' },
  { id: 'mod-details', keys: 'Enter', always: false, group: 'Mods list' },
  { id: 'mod-remove', keys: 'Delete', always: false, group: 'Mods list' },
  ...(
    [
      ['tab-browse', 'Ctrl+1'],
      ['tab-mods', 'Ctrl+2'],
      ['tab-problems', 'Ctrl+3'],
      ['tab-load-order', 'Ctrl+4'],
      ['tab-saves', 'Ctrl+5'],
      ['tab-notes', 'Ctrl+6'],
      ['tab-console', 'Ctrl+7'],
      ['tab-performance', 'Ctrl+8'],
    ] as const
  ).map(([id, keys]) => ({ id, keys, always: false, group: 'Tabs' as const })),
  { id: 'new-profile', keys: 'Ctrl+N', always: false, group: 'Profiles' },
  { id: 'duplicate-profile', keys: 'Ctrl+D', always: false, group: 'Profiles' },
  { id: 'rename-profile', keys: 'F2', always: false, group: 'Profiles' },
  { id: 'find-all-mods', keys: 'Ctrl+Shift+F', always: false, group: 'Mods list' },
  { id: 'import', keys: 'Ctrl+I', always: false, group: 'General' },
  { id: 'export-profile', keys: 'Ctrl+E', always: false, group: 'Profiles' },
  { id: 'downloads', keys: 'Ctrl+J', always: false, group: 'General' },
  { id: 'notifications', keys: 'Ctrl+Shift+N', always: false, group: 'General' },
  { id: 'previous-profile', keys: 'Ctrl+PageUp', always: false, group: 'Navigation' },
  { id: 'next-profile', keys: 'Ctrl+PageDown', always: false, group: 'Navigation' },
  { id: 'collapse-sidebar', keys: 'Ctrl+B', always: false, group: 'Navigation' },
  { id: 'back', keys: 'Alt+Left', always: false, group: 'Navigation' },
  { id: 'help', keys: 'F1', always: false, group: 'General' },
  { id: 'vanilla-play', keys: 'Ctrl+Shift+P', always: false, group: 'General' },
]

export function parseKeys(keys: string): Chord | null {
  const parts = keys.split('+').filter(Boolean)
  if (parts.length === 0) {
    return null
  }
  let ctrlKey = false
  let shiftKey = false
  let altKey = false
  let key = ''
  for (const part of parts) {
    if (part === 'Ctrl' || part === 'Cmd' || part === 'Meta') {
      ctrlKey = true
    } else if (part === 'Shift') {
      shiftKey = true
    } else if (part === 'Alt') {
      altKey = true
    } else {
      key = eventKey(part)
    }
  }
  if (!key) {
    return null
  }
  return { key, ctrlKey, shiftKey, altKey }
}

export function formatChord(e: Chord): string | null {
  if (modifiers.has(e.key) || e.key === '') {
    return null
  }
  const parts: string[] = []
  if (e.ctrlKey || e.metaKey) {
    parts.push('Ctrl')
  }
  if (e.shiftKey) {
    parts.push('Shift')
  }
  if (e.altKey) {
    parts.push('Alt')
  }
  const display = fromEventKey[e.key] ?? (e.key.length === 1 ? e.key.toUpperCase() : e.key)
  parts.push(display)
  return parts.join('+')
}

export function defaultBindings(): Record<ShortcutId, string> {
  return Object.fromEntries(SHORTCUTS.map((row) => [row.id, row.keys])) as Record<
    ShortcutId,
    string
  >
}

export function mergeBindings(stored?: ShortcutBindings | null): Record<ShortcutId, string> {
  const next = defaultBindings()
  if (!stored) {
    return next
  }
  for (const row of SHORTCUTS) {
    const keys = stored[row.id]
    // An empty chord is an action Mortar unbound because the user gave its default to another one.
    if (keys !== undefined) {
      next[row.id] = keys
    }
  }
  return next
}

export function conflictFor(
  id: ShortcutId,
  keys: string,
  bindings: ShortcutBindings,
): ShortcutId | null {
  for (const row of SHORTCUTS) {
    if (row.id !== id && (bindings[row.id] ?? row.keys) === keys) {
      return row.id
    }
  }
  return null
}

export function isTypingTarget(el: TypingTarget | null): boolean {
  if (!el) {
    return false
  }
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') {
    return true
  }
  return Boolean(el.isContentEditable)
}

export function dialogOpen(): boolean {
  return document.querySelector('[role="dialog"]') !== null
}

/** Non-Esc shortcuts stay silent while typing or a dialog is open. */
export function shortcutAllowed(
  id: ShortcutId,
  target: TypingTarget | null,
  dialog = false,
): boolean {
  const row = SHORTCUTS.find((s) => s.id === id)
  if (!row) {
    return false
  }
  if (row.always) {
    return true
  }
  if (isTypingTarget(target) || dialog) {
    return false
  }
  return true
}

export function setShortcutCapturing(on: boolean): void {
  capturing = on
}

export function shortcutCapturing(): boolean {
  return capturing
}

export function matchShortcut(
  e: Chord,
  bindings: ShortcutBindings = defaultBindings(),
): ShortcutId | null {
  const want = chordKey(e)
  for (const row of SHORTCUTS) {
    const parsed = parseKeys(bindings[row.id] ?? row.keys)
    if (parsed && chordKey(parsed) === want) {
      return row.id
    }
  }
  return null
}

/** The shortcut a key event triggers under the user's current bindings. */
export function boundShortcut(
  e: Chord,
  shortcuts: ShortcutBindings | null | undefined,
): ShortcutId | null {
  return matchShortcut(e, mergeBindings(shortcuts))
}
