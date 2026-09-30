import type { SettingsSection } from '../nav/store.ts'
import type { Shortcut, ShortcutId } from '../settings/shortcuts.ts'
import type { PaletteItem } from './match.ts'

export interface PaletteLabels {
  play: string
  updates: string
  downloads: string
  import: string
  share: string
  newProfile: string
  profileHint: string
  modHint: string
  settingsHint: string
}

export function buildPaletteItems(input: {
  profiles: { id: string; name: string }[]
  mods: { key: string; uniqueId: string; name: string }[]
  sections: { id: SettingsSection; label: string }[]
  shortcuts: readonly Shortcut[]
  shortcutLabels: Record<ShortcutId, string>
  labels: PaletteLabels
}): PaletteItem[] {
  const { profiles, mods, sections, shortcuts, shortcutLabels, labels } = input
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
      id: `mod:${mod.key}/${mod.uniqueId}`,
      kind: 'mod',
      label: mod.name,
      hint: `${labels.modHint} · ${mod.uniqueId}`,
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
    { id: 'action:import', kind: 'action', label: labels.import },
    { id: 'action:share', kind: 'action', label: labels.share },
    { id: 'action:new-profile', kind: 'action', label: labels.newProfile },
  )
  for (const row of shortcuts) {
    items.push({
      id: `shortcut:${row.id}`,
      kind: 'shortcut',
      label: shortcutLabels[row.id],
      shortcut: row.keys,
    })
  }
  return items
}
