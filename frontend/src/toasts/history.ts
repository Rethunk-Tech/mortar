export const HISTORY_CAP = 200

export interface HistoryActionState {
  disabled: boolean
  reason?: string
}

export function prependHistory<T>(items: T[], item: T, cap = HISTORY_CAP): T[] {
  return [item, ...items].slice(0, cap)
}

export function changeStillLatest(
  profile: { entries?: { key: string; mods?: { id: string }[] | null }[] | null } | undefined,
  entryKey: string,
  ids: string[],
): HistoryActionState {
  if (!profile) {
    return { disabled: true, reason: 'That profile is gone.' }
  }
  const entries = profile.entries ?? []
  for (const id of ids) {
    const current = entries.find((e) => (e.mods ?? []).some((m) => m.id === id))
    if (current?.key !== entryKey) {
      return { disabled: true, reason: 'This is no longer the latest change to that mod.' }
    }
  }
  return { disabled: false }
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
