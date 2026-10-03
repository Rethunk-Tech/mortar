import type { I18n } from '@lingui/core'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { collectionHeader } from '../game/collectionHeader.ts'
import { useCollectionStatus } from '../game/useCollectionStatus.ts'
import type { SettingsSection } from '../nav/store.ts'
import { userModEntries } from '../profiles/count.ts'
import { SHORTCUTS, type ShortcutId } from '../settings/shortcuts.ts'
import { buildPaletteItems } from './items.ts'
import { matchPaletteItems, type PaletteItem } from './match.ts'
import { paletteActionLabels } from './paletteLabels.ts'

export function usePaletteShown(input: {
  i18n: I18n
  query: string
  profiles: Profile[]
  openId: string | undefined
  gameId: string
  sections: { id: SettingsSection; label: string }[]
  shortcutLabels: Partial<Record<ShortcutId, string>>
  bindings: Record<ShortcutId, string>
}): PaletteItem[] {
  const { i18n, query, profiles, openId, gameId, sections, shortcutLabels, bindings } = input
  const profile = profiles.find((p) => p.id === openId)
  const collectionStatus = useCollectionStatus(gameId, profile)
  const collectionReview = Boolean(
    profile?.collection && collectionHeader(collectionStatus).review !== null,
  )
  const mods = userModEntries(profile?.entries).flatMap((entry) =>
    (entry.mods ?? []).map((mod) => ({
      key: entry.key,
      uniqueId: mod.uniqueId,
      name: mod.name,
    })),
  )
  return matchPaletteItems(
    buildPaletteItems({
      profiles: profiles.map((p) => ({ id: p.id, name: p.name })),
      mods,
      sections,
      shortcuts: SHORTCUTS.map((row) => ({ ...row, keys: bindings[row.id] })),
      shortcutLabels,
      labels: paletteActionLabels(i18n),
      collectionReview,
    }),
    query,
  )
}
