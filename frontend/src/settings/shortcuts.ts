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

export interface Shortcut {
  id: ShortcutId
  keys: string
  always: boolean
}

export interface Chord {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
}

export interface TypingTarget {
  tagName?: string
  isContentEditable?: boolean
}

/** Single table for key handling and Settings › Shortcuts. */
export const SHORTCUTS: readonly Shortcut[] = [
  { id: 'command-palette', keys: 'Ctrl+K', always: false },
  { id: 'filter-mods', keys: 'Ctrl+F', always: false },
  { id: 'play', keys: 'Ctrl+P', always: false },
  { id: 'check-updates', keys: 'F5', always: false },
  { id: 'open-settings', keys: 'Ctrl+,', always: false },
  { id: 'dismiss', keys: 'Esc', always: true },
  { id: 'select-all-mods', keys: 'Ctrl+A', always: false },
  { id: 'mod-up', keys: '↑', always: false },
  { id: 'mod-down', keys: '↓', always: false },
  { id: 'mod-toggle', keys: 'Space', always: false },
  { id: 'mod-details', keys: 'Enter', always: false },
  { id: 'mod-remove', keys: 'Delete', always: false },
]

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

export function matchShortcut(e: Chord): ShortcutId | null {
  const { key, ctrlKey, metaKey } = e
  const ctrl = Boolean(ctrlKey || metaKey)
  if (key === 'Escape') {
    return 'dismiss'
  }
  if (ctrl && (key === 'k' || key === 'K')) {
    return 'command-palette'
  }
  if (ctrl && (key === 'f' || key === 'F')) {
    return 'filter-mods'
  }
  if (ctrl && (key === 'p' || key === 'P')) {
    return 'play'
  }
  if (key === 'F5') {
    return 'check-updates'
  }
  if (ctrl && key === ',') {
    return 'open-settings'
  }
  return null
}
