import { msg } from '@lingui/core/macro'
import { i18n } from '../i18n/index.ts'

export const HISTORY_CAP = 200

export interface HistoryActionState {
  disabled: boolean
  reason?: string
}

export function prependHistory<T>(items: T[], item: T, cap = HISTORY_CAP): T[] {
  return [item, ...items].slice(0, cap)
}

// Why an undo of this change can no longer run, or null while it still can.
export function staleChange(
  profile: { entries?: { key: string; mods?: { id: string }[] | null }[] | null } | undefined,
  entryKey: string,
  ids: string[],
): 'gone' | 'superseded' | null {
  if (!profile) {
    return 'gone'
  }
  const entries = profile.entries ?? []
  const superseded = ids.some(
    (id) => entries.find((e) => (e.mods ?? []).some((m) => m.id === id))?.key !== entryKey,
  )
  return superseded ? 'superseded' : null
}

export function changeStillLatest(
  profile: { entries?: { key: string; mods?: { id: string }[] | null }[] | null } | undefined,
  entryKey: string,
  ids: string[],
): HistoryActionState {
  switch (staleChange(profile, entryKey, ids)) {
    case 'gone':
      return { disabled: true, reason: i18n._(msg`That profile is gone.`) }
    case 'superseded':
      return {
        disabled: true,
        reason: i18n._(msg`This is no longer the latest change to that mod.`),
      }
    default:
      return { disabled: false }
  }
}

export function historyActionState(
  live: (() => HistoryActionState) | undefined,
  locked: boolean,
  lockReason: string,
): HistoryActionState {
  if (locked) {
    return { disabled: true, reason: lockReason }
  }
  return live?.() ?? { disabled: false }
}

/** Events newer than `id`, which reverting to before `id` undoes too. History lists newest first. */
export function laterEvents<T extends { id: string }>(events: readonly T[], id: string): T[] {
  const at = events.findIndex((e) => e.id === id)
  return at < 0 ? [] : events.slice(0, at)
}
