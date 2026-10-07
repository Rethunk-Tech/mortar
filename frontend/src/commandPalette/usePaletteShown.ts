import type { I18n } from '@lingui/core'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { collectionHeader } from '../game/collectionHeader.ts'
import { useCollectionStatus } from '../game/useCollectionStatus.ts'
import type { SettingsSection } from '../nav/store.ts'
import { userModEntries } from '../profiles/count.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { SHORTCUTS, type ShortcutId } from '../settings/shortcuts.ts'
import { useBisectBlockText } from './crashBisect.ts'
import { buildPaletteItems } from './items.ts'
import { paletteActionLabels } from './paletteLabels.ts'
import { arrangePalette, type PaletteRow, readRecent } from './sections.ts'

export function usePaletteShown(input: {
  i18n: I18n
  query: string
  profiles: Profile[]
  openId: string | undefined
  gameId: string
  sections: { id: SettingsSection; label: string }[]
  shortcutLabels: Partial<Record<ShortcutId, string>>
  bindings: Record<ShortcutId, string>
}): PaletteRow[] {
  const { i18n, query, profiles, openId, gameId, sections, shortcutLabels, bindings } = input
  const profile = openProfileOf({ profiles, openId: openId ?? '' })
  const streamOverlay = useProfiles((s) => s.game?.loaders?.[0]?.overlay === true)
  const crashCheckBlocked = useBisectBlockText()
  const collectionStatus = useCollectionStatus(gameId, profile)
  const collectionReview = Boolean(
    profile?.collection && collectionHeader(collectionStatus).review !== null,
  )
  const mods = userModEntries(profile?.entries).flatMap((entry) =>
    (entry.mods ?? []).map((mod) => ({
      key: entry.key,
      id: mod.id,
      name: mod.name,
    })),
  )
  return arrangePalette(
    buildPaletteItems({
      profiles: profiles.map((p) => ({ id: p.id, name: p.name })),
      mods,
      sections,
      shortcuts: SHORTCUTS.map((row) => ({ ...row, keys: bindings[row.id] })),
      shortcutLabels,
      labels: paletteActionLabels(i18n),
      collectionReview,
      streamOverlay,
      crashCheckBlocked,
      profileOpen: profile !== undefined,
    }),
    query,
    readRecent(),
  )
}
