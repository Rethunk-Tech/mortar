import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useEffect, useState } from 'react'
import type { EntrySize } from '../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import { EntrySizes } from '../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { StartupReports } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type {
  CustomCategory,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { modTotal } from '../console/startupView.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { customCategoryById, entryGroupName, resolvedCategoryLabel } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { kindLabel, nexusIdOf, sourceKind } from './lookup.ts'

let entrySizes: Readonly<Record<string, number>> = {}

function useEntrySizes(): Readonly<Record<string, number>> {
  const [sizes, setSizes] = useState<Record<string, number>>({})
  useEffect(() => {
    EntrySizes()
      .then((rows: EntrySize[] | null) => {
        const next: Record<string, number> = {}
        for (const r of rows ?? []) {
          next[r.key] = r.size
        }
        setSizes(next)
        entrySizes = next
      })
      .catch(reportUnexpected)
  }, [])
  entrySizes = sizes
  return sizes
}

let startupCosts: Readonly<Record<string, number>> = {}

/** Each mod's and content pack's share of the latest measured startup, by lower-cased UniqueID. */
function useStartupCosts(game: string, profileId: string): Readonly<Record<string, number>> {
  const [costs, setCosts] = useState<Record<string, number>>({})
  useEffect(() => {
    if (game === '' || profileId === '') {
      return
    }
    StartupReports(game, profileId)
      .then((reports) => {
        const next: Record<string, number> = {}
        for (const mod of reports?.[0]?.mods ?? []) {
          next[mod.id.toLowerCase()] = modTotal(mod)
          for (const pack of mod.packs ?? []) {
            next[pack.id.toLowerCase()] = pack.ms
          }
        }
        setCosts(next)
      })
      .catch(reportUnexpected)
  }, [game, profileId])
  startupCosts = costs
  return costs
}

function toListRow(
  m: Mod,
  profile: Profile,
  byId: Record<number, { details?: ListRow['details'] } | undefined>,
  customCategories: readonly CustomCategory[],
): ListRow {
  const entry = (profile.entries ?? []).find((e) => e.key === m.key)
  const source = kindLabel(sourceKind(profile, m), {
    archive: i18n._(msg`Archive`),
    nexus: i18n._(msg`Nexus Mods`),
    github: i18n._(msg`GitHub`),
  })
  const details = byId[nexusIdOf(profile, m)]?.details
  const customById = customCategoryById(customCategories)
  const categoryLabel = resolvedCategoryLabel(
    entry?.categoryOverride,
    details?.category,
    customById,
  )
  const row: ListRow = {
    mod: m,
    added: entry?.added ?? '',
    pinned: Boolean(entry?.pinned),
    source,
    status: m.enabled ? i18n._(msg`Enabled`) : i18n._(msg`Off`),
    note: entry?.note ?? '',
    tags: [...(entry?.tags ?? [])],
    categoryOverride: entry?.categoryOverride ?? '',
    categoryLabel,
    groupName: entryGroupName(profile.groups, m.key),
  }
  const n = entrySizes[m.key]
  if (n !== undefined) {
    row.size = n
  }
  const ms = startupCosts[m.uniqueId.toLowerCase()]
  if (ms !== undefined) {
    row.startupMs = ms
  }
  if (details) {
    row.details = details
  }
  return row
}

export { toListRow, useEntrySizes, useStartupCosts }
