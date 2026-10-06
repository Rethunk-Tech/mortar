import type { AssetConflict } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type {
  Broken,
  Missing,
  Result,
  Update,
  UpdatesResult,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import {
  Problems,
  Updates,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { HistoryDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ChangesSince } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/service.ts'
import { listNames } from '../i18n/list.ts'
import { localId } from '../mods/dependents.ts'
import { visibleUpdates } from '../mods/lookup.ts'
import { diffLines } from '../profiles/historyDiff.ts'
import { useProfiles } from '../profiles/store.ts'
import { saveName } from '../saves/saveName.ts'

const PLAY_ISSUE_NAME_CAP = 5

const changesSinceCache = new Map<string, HistoryDiff | null>()

type PlayIssueKind =
  | 'missing'
  | 'conflicts'
  | 'updates'
  | 'broken'
  | 'lastProfile'
  | 'changes'
  | 'saveMods'
  | 'saveModsOff'

interface PlayIssueGroup {
  kind: PlayIssueKind
  count: number
  names: string[]
  nameTitles?: string[]
  save?: string
  profileName?: string
  switchProfileId?: string
}

function groupOf(
  kind: PlayIssueKind,
  entries: { name: string; title?: string }[],
): PlayIssueGroup | null {
  if (entries.length === 0) {
    return null
  }
  const slice = entries.slice(0, PLAY_ISSUE_NAME_CAP)
  const titles = slice.map((entry) => entry.title ?? '')
  return {
    kind,
    count: entries.length,
    names: slice.map((entry) => entry.name),
    ...(titles.some((title) => title !== '') ? { nameTitles: titles } : {}),
  }
}

function missingName(m: Missing): { name: string; title?: string } {
  const page = (m.where?.pageName ?? '').trim()
  if (page !== '') {
    return { name: page }
  }
  return { name: 'Unknown mod', title: localId(m.id) }
}

function conflictName(c: AssetConflict): { name: string } {
  const names = (c.names ?? []).filter((n) => n !== '')
  return { name: names.length > 0 ? listNames(names) : c.target }
}

function playIssueSummary(input: {
  missing?: Missing[] | null
  assetConflicts?: AssetConflict[] | null
  broken?: Broken[] | null
  updates?: Update[] | null
  currentProfileId?: string
  lastPlayed?: { folder: string; farm: string; profileId: string; profileName: string } | null
  saveMods?: { name: string; disabled?: boolean }[]
  switchProfileId?: string
}): PlayIssueGroup[] {
  const groups: PlayIssueGroup[] = []
  const missing = groupOf(
    'missing',
    (input.missing ?? []).filter((m) => !(m.optional || m.external)).map(missingName),
  )
  const conflicts = groupOf(
    'conflicts',
    (input.assetConflicts ?? []).filter((c) => !c.cosmetic).map(conflictName),
  )
  const updates = groupOf(
    'updates',
    (input.updates ?? []).map((u) => ({ name: u.name })),
  )
  const broken = groupOf(
    'broken',
    (input.broken ?? [])
      .filter((b) => b.status === 'broken' || b.status === 'obsolete' || b.status === 'cycle')
      .map((b) => ({ name: b.name })),
  )
  for (const g of [missing, conflicts, updates, broken]) {
    if (g) {
      groups.push(g)
    }
  }
  const last = input.lastPlayed
  if (last?.profileId && last.profileId !== input.currentProfileId) {
    groups.push({
      kind: 'lastProfile',
      count: 1,
      names: [],
      save: saveName(last),
      profileName: last.profileName,
      switchProfileId: last.profileId,
    })
  }
  // Mods the profile lacks and mods it has switched off need different fixes, so they are listed apart.
  for (const [kind, disabled] of [
    ['saveMods', false],
    ['saveModsOff', true],
  ] as const) {
    const found = (input.saveMods ?? []).filter((m) => (m.disabled ?? false) === disabled)
    if (found.length > 0) {
      groups.push({
        kind,
        count: found.length,
        names: found.slice(0, PLAY_ISSUE_NAME_CAP).map((m) => m.name),
        ...(input.switchProfileId ? { switchProfileId: input.switchProfileId } : {}),
      })
    }
  }
  return groups
}

async function gatherPlayIssues(game: string, profileId: string): Promise<PlayIssueGroup[]> {
  const [problems, updates, lastFit]: [
    Result,
    UpdatesResult,
    Awaited<ReturnType<typeof LastSaveGap>> | null,
  ] = await Promise.all([
    Problems(game, profileId),
    Updates(game, profileId),
    LastSaveGap(game, profileId).catch(() => null),
  ])
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  const fit = lastFit?.[0]
  const lastName = fit?.lastProfileId
    ? (useProfiles.getState().profiles.find((p) => p.id === fit.lastProfileId)?.name ??
      fit.lastProfileId)
    : ''
  const groups = playIssueSummary({
    missing: problems.missing,
    assetConflicts: problems.assetConflicts,
    broken: problems.broken,
    updates: visibleUpdates(updates, profile),
    currentProfileId: profileId,
    lastPlayed:
      fit?.lastProfileId && lastName
        ? {
            folder: fit.folder,
            farm: fit.farm,
            profileId: fit.lastProfileId,
            profileName: lastName,
          }
        : null,
    saveMods: (fit?.lastMissing ?? []).map((m) => ({
      name: m.name || localId(m.id),
      disabled: m.disabled,
    })),
    switchProfileId: fit?.lastProfileExists ? fit.lastProfileId : '',
  })
  try {
    const runs = await Runs(game, profileId)
    const since = runs?.[0]?.started ?? ''
    const key = `${game}\0${profileId}\0${profile?.updated ?? ''}\0${since}`
    let diff: HistoryDiff | null | undefined
    if (changesSinceCache.has(key)) {
      diff = changesSinceCache.get(key)
    } else {
      diff = await ChangesSince(game, profileId, since)
      changesSinceCache.set(key, diff)
    }
    const names = diff ? diffLines(diff) : []
    if (names.length > 0) {
      groups.push({
        kind: 'changes',
        count: names.length,
        names: names.slice(0, PLAY_ISSUE_NAME_CAP),
      })
    }
  } catch {
    // pre-Play still shows problems when history is unavailable
  }
  return groups
}

function overflowIssueCount(group: Pick<PlayIssueGroup, 'count' | 'names'>): number {
  return Math.max(0, group.count - group.names.length)
}

export type { PlayIssueGroup }
export { gatherPlayIssues, overflowIssueCount, playIssueSummary }
