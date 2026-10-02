const tabNumberKey = /^[1-6]$/

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
  | 'tab-mods'
  | 'tab-problems'
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
  group: 'General' | 'Navigation' | 'Profiles' | 'Mods list' | 'Console'
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
      ['tab-mods', 'Ctrl+1'],
      ['tab-problems', 'Ctrl+2'],
      ['tab-saves', 'Ctrl+3'],
      ['tab-notes', 'Ctrl+4'],
      ['tab-console', 'Ctrl+5'],
      ['tab-performance', 'Ctrl+6'],
    ] as const
  ).map(([id, keys]) => ({ id, keys, always: false, group: 'Console' as const })),
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
  const { key, ctrlKey, metaKey, shiftKey, altKey } = e
  const ctrl = Boolean(ctrlKey || metaKey)
  if (key === 'Escape') {
    return 'dismiss'
  }
  if (altKey && key === 'ArrowLeft') {
    return 'back'
  }
  const fixed: Record<string, ShortcutId> = {
    F1: 'help',
    F2: 'rename-profile',
    F5: 'check-updates',
    ArrowUp: 'mod-up',
    ArrowDown: 'mod-down',
    ' ': 'mod-toggle',
    Enter: 'mod-details',
    Delete: 'mod-remove',
  }
  if (!ctrl) {
    return fixed[key] ?? null
  }
  if (shiftKey) {
    return (
      ({ f: 'find-all-mods', n: 'notifications', p: 'vanilla-play' } as const)[
        key.toLowerCase() as 'f' | 'n' | 'p'
      ] ?? null
    )
  }
  if (tabNumberKey.test(key)) {
    return `tab-${['mods', 'problems', 'saves', 'notes', 'console', 'performance'][Number(key) - 1]}` as ShortcutId
  }
  return (
    (
      {
        k: 'command-palette',
        f: 'filter-mods',
        p: 'play',
        n: 'new-profile',
        d: 'duplicate-profile',
        i: 'import',
        e: 'export-profile',
        j: 'downloads',
        b: 'collapse-sidebar',
        a: 'select-all-mods',
        ',': 'open-settings',
        PageUp: 'previous-profile',
        PageDown: 'next-profile',
      } as Record<string, ShortcutId>
    )[key.length === 1 ? key.toLowerCase() : key] ?? null
  )
}
