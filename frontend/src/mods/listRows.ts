import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useEffect, useState } from 'react'
import type { EntrySize } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import { EntrySizes } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { StartupReports } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PackageOverrides } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { modTotal } from '../console/startupView.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { idKey } from './dependents.ts'
import { type customCategoryById, resolvedCategoryLabel } from './group.ts'
import type { ListRow } from './listColumns.ts'
import { kindLabel } from './lookup.ts'

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
      })
      .catch(reportUnexpected)
  }, [])
  return sizes
}

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
          next[idKey(mod.id)] = modTotal(mod)
          for (const pack of mod.packs ?? []) {
            next[idKey(pack.id)] = pack.ms
          }
        }
        setCosts(next)
      })
      .catch(reportUnexpected)
  }, [game, profileId])
  return costs
}

/** Files each package wins over earlier ones, by entry key, for a game whose profile is deployed by linking. */
function usePackageOverrides(
  game: string,
  profile: Profile,
  deploy: string,
): Readonly<Record<string, number | undefined>> {
  const [wins, setWins] = useState<Readonly<Record<string, number | undefined>>>({})
  useEffect(() => {
    if (deploy !== 'profile' || game === '') {
      return
    }
    PackageOverrides(game, profile.id)
      .then((r) => setWins(r ?? {}))
      .catch(reportUnexpected)
  }, [game, profile, deploy])
  return wins
}

// One pass over the profile serves every row: lookups by key, not a scan of the entries per mod.
function toListRows(
  mods: readonly Mod[],
  profile: Profile,
  data: {
    byId: Record<number, { details?: ListRow['details'] } | undefined>
    customById: ReturnType<typeof customCategoryById>
    sizes: Readonly<Record<string, number>>
    costs: Readonly<Record<string, number>>
    wins: Readonly<Record<string, number | undefined>>
  },
): ListRow[] {
  const entries = new Map((profile.entries ?? []).map((e) => [e.key, e]))
  const groupOf = new Map<string, string>()
  for (const g of profile.groups ?? []) {
    for (const key of g.keys ?? []) {
      if (!groupOf.has(key)) {
        groupOf.set(key, g.name ?? '')
      }
    }
  }
  const place = new Map((profile.entries ?? []).map((e, i) => [e.key, i + 1]))
  return mods.map((m) =>
    toListRow(
      m,
      entries.get(m.key),
      { groupName: groupOf.get(m.key) ?? '', order: place.get(m.key) ?? 0 },
      data,
    ),
  )
}

function toListRow(
  m: Mod,
  entry: Entry | undefined,
  { groupName, order }: { groupName: string; order: number },
  { byId, customById, sizes, costs, wins }: Parameters<typeof toListRows>[2],
): ListRow {
  const source = kindLabel(entry?.source.kind ?? '', {
    archive: i18n._(msg`Archive`),
    nexus: i18n._(msg`Nexus Mods`),
    github: i18n._(msg`GitHub`),
  })
  const nexusId = entry?.source.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  const details = byId[nexusId]?.details
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
    groupName,
    order,
  }
  const won = wins[m.key]
  if (won) {
    row.overrides = won
  }
  const n = sizes[m.key]
  if (n !== undefined) {
    row.size = n
  }
  const ms = costs[idKey(m.id)]
  if (ms !== undefined) {
    row.startupMs = ms
  }
  if (details) {
    row.details = details
  }
  return row
}

export { toListRows, useEntrySizes, usePackageOverrides, useStartupCosts }
