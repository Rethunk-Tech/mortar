export const HISTORY_CAP = 200

export interface HistoryActionState {
  disabled: boolean
  reason?: string
}

export function prependHistory<T>(items: T[], item: T, cap = HISTORY_CAP): T[] {
  return [item, ...items].slice(0, cap)
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
