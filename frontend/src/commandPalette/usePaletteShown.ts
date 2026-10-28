import type { I18n } from '@lingui/core'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { collectionHeader } from '../game/collectionHeader.ts'
import { tabUnavailable } from '../game/sidebarTabs.ts'
import type { TabId } from '../game/tab.ts'
import { useCollectionStatus } from '../game/useCollectionStatus.ts'
import { useGameInfo } from '../games/info.ts'
import { useConfigList } from '../mods/typedConfig/configList.ts'
import type { SettingsSection } from '../nav/store.ts'
import { userModEntries } from '../profiles/count.ts'
import { openProfileOf, useProfileLoader, useProfiles } from '../profiles/store.ts'
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
  const loader = useProfileLoader()
  const gameInfo = useGameInfo()
  const caps = {
    order: Boolean(loader?.order),
    console: Boolean(loader?.console),
    startup: Boolean(loader?.startup),
    screenshots: Boolean(gameInfo?.hasScreenshots),
    saves: Boolean(gameInfo?.hasSaves),
    develop: Boolean(gameInfo?.loaders?.[0]?.devTemplate),
  }
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
  const configList = useConfigList((s) => s.byProfile[openId ?? '']?.list)
  const configurable = new Set((configList?.mods ?? []).map((m) => m.id))
  return arrangePalette(
    buildPaletteItems({
      profiles: profiles.map((p) => ({ id: p.id, name: p.name })),
      mods,
      configurable,
      sections,
      shortcuts: SHORTCUTS.map((row) => ({ ...row, keys: bindings[row.id] })),
      shortcutLabels,
      labels: paletteActionLabels(i18n),
      collectionReview,
      streamOverlay,
      crashCheckBlocked,
      tabHidden: (tab) => loader !== undefined && tabUnavailable(tab as TabId, caps),
      profileOpen: profile !== undefined,
    }),
    query,
    readRecent(),
  )
}
