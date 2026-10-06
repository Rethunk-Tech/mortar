import { localId } from '../mods/dependents.ts'
import type { SettingsSection } from '../nav/store.ts'
import type { Shortcut, ShortcutId } from '../settings/shortcuts.ts'
import type { PaletteItem } from './match.ts'

export interface PaletteLabels {
  play: string
  updates: string
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
  } = input
  const items: PaletteItem[] = []
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
    { id: 'action:play', kind: 'action', label: labels.play },
    { id: 'action:updates', kind: 'action', label: labels.updates },
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
    { id: 'action:find-crash-cause', kind: 'action', label: labels.findCrashCause },
    { id: 'action:share', kind: 'action', label: labels.share },
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
    })),
  )
  for (const row of shortcuts) {
    items.push({
      id: `shortcut:${row.id}`,
      kind: 'shortcut',
      label: shortcutLabels[row.id] ?? row.keys,
      shortcut: row.keys,
    })
  }
  return items
}
