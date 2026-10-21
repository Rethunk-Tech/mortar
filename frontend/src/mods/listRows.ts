import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useEffect } from 'react'
import { create } from 'zustand'
import type { EntrySize } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import { EntrySizes } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { StartupReports } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type {
  Entry,
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PackageOverrides } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { modTotal } from '../console/startupView.ts'
import { sameValue } from '../shell/sameValue.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { idKey } from './dependents.ts'
import { type customCategoryById, resolvedCategoryLabel } from './group.ts'
import { partKey, supportParts } from './installReason.ts'
import type { ListRow } from './listColumns.ts'
import { kindLabel } from './lookup.ts'
import { siteCategory } from './modSearch.ts'

// Fetched again each time the mods list mounts, but kept between mounts and replaced only when an answer differs, so
// switching views or tabs does not first draw every row without them and then again with them.
const useListData = create<{
  sizes: Readonly<Record<string, number>>
  costs: Readonly<Record<string, Readonly<Record<string, number>>>>
  wins: Readonly<Record<string, Readonly<Record<string, number | undefined>>>>
}>(() => ({ sizes: {}, costs: {}, wins: {} }))
const NONE: Readonly<Record<string, never>> = {}

function keyed<T>(field: 'costs' | 'wins', key: string, next: T) {
  useListData.setState((s) =>
    sameValue(s[field][key], next) ? s : { [field]: { ...s[field], [key]: next } },
  )
}

function useEntrySizes(): Readonly<Record<string, number>> {
  useEffect(() => {
    EntrySizes()
      .then((rows: EntrySize[] | null) => {
        const next: Record<string, number> = {}
        for (const r of rows ?? []) {
          next[r.key] = r.size
        }
        useListData.setState((s) => (sameValue(s.sizes, next) ? s : { sizes: next }))
      })
      .catch(reportUnexpected)
  }, [])
  return useListData((s) => s.sizes)
}

/** Each mod's and content pack's share of the latest measured startup, by lower-cased UniqueID. */
function useStartupCosts(game: string, profileId: string): Readonly<Record<string, number>> {
  const key = `${game}/${profileId}`
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
        keyed('costs', `${game}/${profileId}`, next)
      })
      .catch(reportUnexpected)
  }, [game, profileId])
  return useListData((s) => s.costs[key]) ?? NONE
}

/** Files each package wins over earlier ones, by entry key, for a game whose profile is deployed by linking. */
function usePackageOverrides(
  game: string,
  profile: Profile,
  deploy: string,
): Readonly<Record<string, number | undefined>> {
  const key = `${game}/${profile.id}`
  useEffect(() => {
    if (deploy !== 'profile' || game === '') {
      return
    }
    PackageOverrides(game, profile.id)
      .then((r) => keyed('wins', `${game}/${profile.id}`, r ?? {}))
      .catch(reportUnexpected)
  }, [game, profile, deploy])
  return useListData((s) => s.wins[key]) ?? NONE
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
  const parts = supportParts(mods)
  return mods.map((m) =>
    toListRow(
      m,
      entries.get(m.key),
      {
        groupName: groupOf.get(m.key) ?? '',
        order: place.get(m.key) ?? 0,
        implicit: Boolean(entries.get(m.key)?.implicit) || parts.has(partKey(m)),
      },
      data,
    ),
  )
}

function toListRow(
  m: Mod,
  entry: Entry | undefined,
  { groupName, order, implicit }: { groupName: string; order: number; implicit: boolean },
  { byId, customById, sizes, costs, wins }: Parameters<typeof toListRows>[2],
): ListRow {
  const pageless = entry?.source.kind === 'nexus' && (entry.source.modId ?? 0) <= 0
  const known = kindLabel(entry?.source.kind ?? '', {
    archive: i18n._(msg`Archive`),
    nexus: i18n._(msg`Nexus Mods`),
    github: sourceLabel('github'),
    thunderstore: sourceLabel('thunderstore'),
  })
  const fileless = entry?.source.kind === 'nexus' && !pageless && (entry.source.fileId ?? 0) <= 0
  const unknownFile = fileless ? i18n._(msg`Nexus Mods, file unknown`) : known
  const source = pageless ? i18n._(msg`Unknown source`) : unknownFile
  const nexusId = entry?.source.kind === 'nexus' ? (entry.source.modId ?? 0) : 0
  const details = byId[nexusId]?.details
  const categoryLabel = resolvedCategoryLabel(
    entry?.categoryOverride,
    siteCategory(entry, details?.category),
    customById,
  )
  const row: ListRow = {
    mod: m,
    added: entry?.added ?? '',
    switchedOff: m.enabled ? '' : (entry?.switchedOff?.[m.id] ?? ''),
    pinned: Boolean(entry?.pinned),
    implicit,
    reason: implicit ? i18n._(msg`Implicit`) : i18n._(msg`Explicit`),
    source,
    unsourced: pageless || fileless || entry?.source.kind === 'local',
    status: m.enabled ? i18n._(msg`Enabled`) : i18n._(msg`Disabled`),
    note: entry?.note ?? '',
    tags: [...(entry?.tags ?? [])],
    categoryOverride: entry?.categoryOverride ?? '',
    categoryLabel,
    siteCategory: entry?.source.kind === 'thunderstore' ? entry.source.category : undefined,
    groupName,
    download: entry?.source.modName || entry?.source.repo || entry?.source.name || '',
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
