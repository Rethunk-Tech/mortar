import { localId } from '../mods/dependents.ts'
import type { SettingsSection } from '../nav/store.ts'
import type { Shortcut, ShortcutId } from '../settings/shortcuts.ts'
import type { PaletteItem } from './match.ts'

// A shortcut that does what a palette action does shows its keys on that action instead of as a second row.
const ACTION_OF_SHORTCUT: Partial<Record<ShortcutId, string>> = {
  play: 'action:play',
  'new-profile': 'action:new-profile',
  import: 'action:import',
  downloads: 'action:downloads',
  'export-profile': 'action:share',
  'tab-browse': 'tab:browse',
  'tab-mods': 'tab:mods',
  'tab-problems': 'tab:problems',
  'tab-load-order': 'tab:load-order',
  'tab-saves': 'tab:saves',
  'tab-config': 'tab:config',
  'tab-console': 'tab:console',
  'tab-performance': 'tab:performance',
}

// Shortcuts that act on the focused list or the window itself, which the palette has no focus to run.
const KEYBOARD_ONLY = new Set<ShortcutId>([
  'command-palette',
  'dismiss',
  'select-all-mods',
  'mod-up',
  'mod-down',
  'mod-toggle',
  'mod-details',
  'mod-remove',
  'filter-mods',
])

// Rows that need an open profile to do anything.
const PROFILE_ONLY = new Set([
  'action:play',
  'action:share',
  'shortcut:duplicate-profile',
  'shortcut:rename-profile',
  'shortcut:vanilla-play',
  ...[
    'home',
    'browse',
    'mods',
    'problems',
    'load-order',
    'saves',
    'config',
    'console',
    'performance',
  ].map((tab) => `tab:${tab}`),
])

export interface PaletteLabels {
  play: string
  downloads: string
  downloadsFolder: string
  import: string
  pasteLink: string
  collectionReview: string
  findCrashCause: string
  configureMod: (name: string) => string
  share: string
  newProfile: string
  streamOverlay: string
  recentChanges: string
  diagnostics: string
  profileHint: string
  modHint: string
  settingsHint: string
  // Why an action that works on the open profile cannot run while none is open.
  needProfile: string
  tabs: Record<string, string>
  toggle: (name: string) => string
}

export function buildPaletteItems(input: {
  profiles: { id: string; name: string }[]
  mods: { key: string; id: string; name: string }[]
  sections: { id: SettingsSection; label: string }[]
  shortcuts: readonly Shortcut[]
  shortcutLabels: Partial<Record<ShortcutId, string>>
  labels: PaletteLabels
  collectionReview?: boolean
  streamOverlay?: boolean
  crashCheckBlocked?: string | null
  profileOpen: boolean
}): PaletteItem[] {
  const {
    profiles,
    mods,
    sections,
    shortcuts,
    shortcutLabels,
    labels,
    collectionReview,
    streamOverlay,
    crashCheckBlocked,
    profileOpen,
  } = input
  const items: PaletteItem[] = []
  const needsProfile = (id: string) =>
    PROFILE_ONLY.has(id) && !profileOpen
      ? { hint: labels.needProfile, disabled: labels.needProfile }
      : {}
  for (const profile of profiles) {
    items.push({
      id: `profile:${profile.id}`,
      kind: 'profile',
      label: profile.name,
      hint: labels.profileHint,
    })
  }
  for (const mod of mods) {
    items.push({
      id: `mod:${mod.key}/${mod.id}`,
      kind: 'mod',
      label: mod.name,
      hint: labels.modHint,
      match: localId(mod.id),
    })
    items.push({
      id: `toggle-mod:${mod.key}/${mod.id}`,
      kind: 'action',
      label: labels.toggle(mod.name),
      hint: labels.modHint,
      match: localId(mod.id),
    })
    items.push({
      id: `configure-mod:${mod.key}/${mod.id}`,
      kind: 'action',
      label: labels.configureMod(mod.name),
      hint: labels.modHint,
      match: localId(mod.id),
    })
  }
  for (const section of sections) {
    items.push({
      id: `settings:${section.id}`,
      kind: 'settings',
      label: section.label,
      hint: labels.settingsHint,
    })
  }
  items.push(
    { id: 'action:play', kind: 'action', label: labels.play, ...needsProfile('action:play') },
    { id: 'action:downloads', kind: 'action', label: labels.downloads },
    { id: 'action:downloads-folder', kind: 'action', label: labels.downloadsFolder },
    { id: 'action:import', kind: 'action', label: labels.import },
    { id: 'action:paste-link', kind: 'action', label: labels.pasteLink },
    ...(collectionReview
      ? [
          {
            id: 'action:collection-review',
            kind: 'action' as const,
            label: labels.collectionReview,
          },
        ]
      : []),
    {
      id: 'action:find-crash-cause',
      kind: 'action',
      label: labels.findCrashCause,
      ...(crashCheckBlocked ? { hint: crashCheckBlocked, disabled: crashCheckBlocked } : {}),
    },
    { id: 'action:share', kind: 'action', label: labels.share, ...needsProfile('action:share') },
    { id: 'action:new-profile', kind: 'action', label: labels.newProfile },
    ...(streamOverlay
      ? [{ id: 'action:stream-overlay', kind: 'action' as const, label: labels.streamOverlay }]
      : []),
    { id: 'action:recent-changes', kind: 'action', label: labels.recentChanges },
    { id: 'settings:about', kind: 'settings', label: labels.diagnostics },
    ...Object.entries(labels.tabs).map(([id, label]) => ({
      id: `tab:${id}`,
      kind: 'action' as const,
      label,
      ...needsProfile(`tab:${id}`),
    })),
  )
  for (const row of shortcuts) {
    const action = ACTION_OF_SHORTCUT[row.id]
    const merged = action ? items.find((item) => item.id === action) : undefined
    if (merged) {
      merged.shortcut = row.keys
    } else if (!KEYBOARD_ONLY.has(row.id)) {
      items.push({
        id: `shortcut:${row.id}`,
        kind: 'shortcut',
        label: shortcutLabels[row.id] ?? row.keys,
        shortcut: row.keys,
        ...needsProfile(`shortcut:${row.id}`),
      })
    }
  }
  return items
}
