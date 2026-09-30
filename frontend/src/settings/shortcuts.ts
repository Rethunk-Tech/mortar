export type ShortcutId = 'filter-mods' | 'play' | 'check-updates' | 'open-settings' | 'dismiss'

export interface Shortcut {
  id: ShortcutId
  keys: string
  label: string
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
  { id: 'filter-mods', keys: 'Ctrl+F', label: 'Focus the Mods filter', always: false },
  { id: 'play', keys: 'Ctrl+P', label: 'Play the open profile', always: false },
  { id: 'check-updates', keys: 'F5', label: 'Check for mod updates', always: false },
  { id: 'open-settings', keys: 'Ctrl+,', label: 'Open Settings', always: false },
  { id: 'dismiss', keys: 'Esc', label: 'Close dialog or clear selection', always: true },
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
