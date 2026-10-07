import { plural } from '@lingui/core/macro'
import {
  HistoryChange,
  type HistoryEvent,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { historyLabel } from './historyLabel.ts'

const KIND_OF: Partial<Record<HistoryChange, HistoryKind>> = {
  [HistoryChange.ChangeAdded]: 'added',
  [HistoryChange.ChangeImported]: 'added',
  [HistoryChange.ChangeMoved]: 'added',
  [HistoryChange.ChangeRemoved]: 'removed',
  [HistoryChange.ChangeUpdated]: 'updated',
  [HistoryChange.ChangeChannel]: 'updated',
  [HistoryChange.ChangeEnabled]: 'enabled',
  [HistoryChange.ChangeDisabled]: 'disabled',
  [HistoryChange.ChangeKnownGood]: 'knownGood',
  [HistoryChange.ChangeSettings]: 'settings',
  [HistoryChange.ChangeLaunch]: 'settings',
  [HistoryChange.ChangeLoader]: 'settings',
  [HistoryChange.ChangeInstall]: 'settings',
  [HistoryChange.ChangeSaves]: 'settings',
  [HistoryChange.ChangeConfigEdited]: 'settings',
  [HistoryChange.ChangeConfigReset]: 'settings',
  [HistoryChange.ChangeOptionSet]: 'settings',
  [HistoryChange.ChangePresetApplied]: 'settings',
}

const dayKey = (d: Date): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

export type HistoryKind =
  | 'added'
  | 'removed'
  | 'updated'
  | 'enabled'
  | 'disabled'
  | 'settings'
  | 'knownGood'
  | 'other'

// Which icon an entry gets.
export const historyKind = (ev: Pick<HistoryEvent, 'change'>): HistoryKind =>
  KIND_OF[ev.change] ?? 'other'

// One sentence for an entry. A change that touched several mods says how many of each kind; the rest use the
// event's own wording.
export function historySummary(
  ev: Parameters<typeof historyLabel>[0] & Partial<HistoryEvent>,
): string {
  const added = ev.added ?? 0
  const removed = ev.removed ?? 0
  const updated = ev.updated ?? 0
  if (ev.change !== HistoryChange.ChangeMods || added + removed + updated === 0) {
    return historyLabel(ev)
  }
  const parts: string[] = []
  if (added > 0) {
    parts.push(plural(added, { one: 'Added # mod', other: 'Added # mods' }))
  }
  if (removed > 0) {
    parts.push(plural(removed, { one: 'Removed # mod', other: 'Removed # mods' }))
  }
  if (updated > 0) {
    parts.push(plural(updated, { one: 'Updated # mod', other: 'Updated # mods' }))
  }
  return parts.join(' · ')
}

export const sentence = (text: string): string => text.charAt(0).toUpperCase() + text.slice(1)

export interface HistoryDay<T> {
  // The local calendar day, as YYYY-MM-DD.
  day: string
  label: 'today' | 'yesterday' | 'date'
  // The day's first event time, for wording a plain date.
  at: string
  events: T[]
}

// Newest-first events grouped by local day, each group saying whether it is today, yesterday or an older date.
export function historyDays<T extends { at: string | Date }>(
  events: readonly T[],
  now: Date = new Date(),
): HistoryDay<T>[] {
  const today = dayKey(now)
  const yesterday = dayKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1))
  const days: HistoryDay<T>[] = []
  for (const ev of events) {
    const day = dayKey(new Date(ev.at))
    const last = days.at(-1)
    if (last?.day === day) {
      last.events.push(ev)
    } else {
      let label: HistoryDay<T>['label'] = 'date'
      if (day === today) {
        label = 'today'
      } else if (day === yesterday) {
        label = 'yesterday'
      }
      days.push({ day, label, at: new Date(ev.at).toISOString(), events: [ev] })
    }
  }
  return days
}
