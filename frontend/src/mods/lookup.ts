import type {
  AssetConflict,
  Broken,
  Copy,
  Duplicate,
  Missing,
  Result,
  RunError,
  SettingHint,
  Update,
  UpdatesResult,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { gamePrefs } from '../settings/gamePrefs.ts'
import { useSettings } from '../settings/store.ts'
import { offersNexusDownload } from './nexusMark.ts'

export const siblingsOf = (mods: Mod[], mod: Mod) =>
  mods.filter((m) => m.key === mod.key && m.uniqueId !== mod.uniqueId)

export const entryOf = (profile: Profile | null | undefined, key: string) =>
  (profile?.entries ?? []).find((e) => e.key === key)

export const sourceKind = (profile: Profile, mod: Mod) =>
  entryOf(profile, mod.key)?.source.kind ?? ''

// The Nexus mod ID a mod was installed from, or 0 for any other source.
export const nexusIdOf = (profile: Profile, mod: Mod) => {
  const source = (profile.entries ?? []).find((e) => e.key === mod.key)?.source
  return source?.kind === 'nexus' ? (source.modId ?? 0) : 0
}

export const kindLabel = (
  kind: string,
  labels: { archive: string; nexus: string; github: string },
) => {
  switch (kind) {
    case 'local':
      return labels.archive
    case 'nexus':
      return labels.nexus
    case 'github':
      return labels.github
    default:
      return kind
  }
}

// A mod is told apart by its entry too, since two entries can hold the same UniqueID.
export const modId = (mod: Pick<Mod, 'key' | 'uniqueId'>) => `${mod.key}/${mod.uniqueId}`

// Showing the mod already open keeps what the panel loaded: it will not load again, since the mod is unchanged.
export const reshow = <E extends { id: string }>(
  s: { detailId: string; extras: E | null },
  mod: Pick<Mod, 'key' | 'uniqueId'> | null,
) => {
  const detailId = mod ? modId(mod) : ''
  return { detailId, extras: detailId === s.detailId ? s.extras : null }
}

export type Problem =
  | { kind: 'broken'; broken: Broken }
  | { kind: 'missing'; missing: Missing }
  | { kind: 'duplicate'; duplicate: Duplicate }
  | { kind: 'asset'; asset: AssetConflict }
  | { kind: 'runError'; runError: RunError }
  | { kind: 'setting'; setting: SettingHint }

export const problemsOf = (result: Result | null): Problem[] =>
  result
    ? [
        ...(result.duplicates ?? []).map(
          (duplicate): Problem => ({ kind: 'duplicate', duplicate }),
        ),
        ...(result.broken ?? []).map((broken): Problem => ({ kind: 'broken', broken })),
        ...(result.missing ?? []).map((missing): Problem => ({ kind: 'missing', missing })),
        // Cosmetic conflicts are listed on the Problems tab only; they are never a problem to count or fix.
        ...(result.assetConflicts ?? [])
          .filter((asset) => !asset.cosmetic)
          .map((asset): Problem => ({ kind: 'asset', asset })),
        ...(result.runErrors ?? []).map((runError): Problem => ({ kind: 'runError', runError })),
        ...(result.settings ?? []).map((setting): Problem => ({ kind: 'setting', setting })),
      ]
    : []

export const entryHasDrift = (result: Result | null, key: string): boolean =>
  (result?.drift ?? []).some((d) => d.kind !== 'unknown' && d.key === key)

export const modStatusProblem = (result: Result | null, mod: Mod): boolean =>
  problemsOf(result).some((p) => concerns(p, mod)) || entryHasDrift(result, mod.key)

export const sameId = (a: string, b: string) => a.toLowerCase() === b.toLowerCase()

// A card is flagged for a broken mod, for each copy of a duplicate, and for the dependent of a missing dependency.
export function concerns(p: Problem, mod: Mod): boolean {
  if (p.kind === 'broken') {
    return p.broken.key === mod.key && sameId(p.broken.uniqueId, mod.uniqueId)
  }
  if (p.kind === 'duplicate') {
    return (
      sameId(p.duplicate.uniqueId, mod.uniqueId) &&
      (p.duplicate.copies ?? []).some((c) => c.key === mod.key)
    )
  }
  if (p.kind === 'asset') {
    return (p.asset.packIds ?? []).some((id) => sameId(id, mod.uniqueId))
  }
  if (p.kind === 'runError') {
    return p.runError.key === mod.key && sameId(p.runError.uniqueId, mod.uniqueId)
  }
  if (p.kind === 'setting') {
    return p.setting.key === mod.key && sameId(p.setting.uniqueId, mod.uniqueId)
  }
  return sameId(p.missing.dependentId, mod.uniqueId)
}

// The copy to keep unless the user picks another: the newest, and among equals the Nexus-sourced one.
export function preselect(copies: Copy[]): string {
  const newest = copies.filter((c) => c.newest)
  const pool = newest.length > 0 ? newest : copies
  return (pool.find((c) => c.nexus) ?? pool[0])?.key ?? ''
}

export function nexusKeepKey(copies: Copy[]): string | null {
  const nexus = copies.filter((c) => c.nexus)
  return nexus.length === 1 ? (nexus[0]?.key ?? null) : null
}

export const problemCount = (result: Result | null): number =>
  problemsOf(result).length + (result?.drift?.length ?? 0)

export const offersUpdate = (
  entry: { pinned?: boolean; skipVersion?: string; skipSources?: string[] | null } | undefined,
  newer: string,
  source = '',
): boolean => {
  if (!newer || entry?.pinned) {
    return false
  }
  if (source !== '' && (entry?.skipSources ?? []).includes(source)) {
    return false
  }
  const skip = entry?.skipVersion ?? ''
  return skip === '' || skip !== newer
}

export const visibleUpdates = (result: UpdatesResult | null, profile?: Profile | null): Update[] =>
  (result?.updates ?? []).filter((u) => {
    if (gamePrefs(useSettings.getState()).smapiBuilds === 'never' && u.unofficial) {
      return false
    }
    return offersUpdate(entryOf(profile, u.key), u.version, u.source)
  })

export const listedAgainstNexus = (
  u: Update,
  page?: { status?: string; available?: boolean },
): boolean => {
  if (u.nexusId > 0 && page && !offersNexusDownload(page.status, page.available)) {
    return false
  }
  return true
}

export const updatesForReview = (
  result: UpdatesResult | null,
  profile?: Profile | null,
  details: Record<
    number,
    { details?: { page?: { status?: string; available?: boolean } } } | undefined
  > = {},
): Update[] =>
  visibleUpdates(result, profile).filter((u) =>
    listedAgainstNexus(u, details[u.nexusId]?.details?.page),
  )

export const installableUpdate = (u: Update): boolean =>
  (!u.unofficial || gamePrefs(useSettings.getState()).smapiBuilds === 'include') &&
  (u.githubRepo !== '' || u.nexusId > 0)

export const updateCount = (
  result: UpdatesResult | null,
  profile?: Profile | null,
  details?: Record<
    number,
    { details?: { page?: { status?: string; available?: boolean } } } | undefined
  >,
): number => updatesForReview(result, profile, details).length

// The update SMAPI's API suggests for this very copy of a mod, if any.
export const updateFor = (
  result: UpdatesResult | null,
  mod: Pick<Mod, 'key' | 'uniqueId'>,
  profile?: Profile | null,
): Update | undefined =>
  visibleUpdates(result, profile).find((u) => u.key === mod.key && sameId(u.uniqueId, mod.uniqueId))
