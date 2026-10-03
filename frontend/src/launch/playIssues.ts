import type {
  AssetConflict,
  Broken,
  Missing,
  Result,
  Update,
  UpdatesResult,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  Problems,
  Updates,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { LastSaveGap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/service.ts'
import { visibleUpdates } from '../mods/lookup.ts'
import { useProfiles } from '../profiles/store.ts'

const PLAY_ISSUE_NAME_CAP = 5

type PlayIssueKind = 'missing' | 'conflicts' | 'updates' | 'broken' | 'lastProfile'

interface PlayIssueGroup {
  kind: PlayIssueKind
  count: number
  names: string[]
  save?: string
  profileName?: string
  switchProfileId?: string
}

function groupOf(kind: PlayIssueKind, labels: string[]): PlayIssueGroup | null {
  if (labels.length === 0) {
    return null
  }
  return { kind, count: labels.length, names: labels.slice(0, PLAY_ISSUE_NAME_CAP) }
}

function missingName(m: Missing): string {
  return m.where?.pageName || m.uniqueId
}

function conflictName(c: AssetConflict): string {
  const names = (c.names ?? []).filter((n) => n !== '')
  return names.length > 0 ? names.join(', ') : c.target
}

function playIssueSummary(input: {
  missing?: Missing[] | null
  assetConflicts?: AssetConflict[] | null
  broken?: Broken[] | null
  updates?: Update[] | null
  currentProfileId?: string
  lastPlayed?: { folder: string; farm: string; profileId: string; profileName: string } | null
}): PlayIssueGroup[] {
  const groups: PlayIssueGroup[] = []
  const missing = groupOf(
    'missing',
    (input.missing ?? []).filter((m) => !m.optional).map(missingName),
  )
  const conflicts = groupOf(
    'conflicts',
    (input.assetConflicts ?? []).filter((c) => !c.cosmetic).map(conflictName),
  )
  const updates = groupOf(
    'updates',
    (input.updates ?? []).map((u) => u.name),
  )
  const broken = groupOf(
    'broken',
    (input.broken ?? [])
      .filter((b) => b.status === 'broken' || b.status === 'obsolete')
      .map((b) => b.name),
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
      save: last.farm || last.folder,
      profileName: last.profileName,
      switchProfileId: last.profileId,
    })
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
  return playIssueSummary({
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
  })
}

export type { PlayIssueGroup, PlayIssueKind }
export { gatherPlayIssues, playIssueSummary }
