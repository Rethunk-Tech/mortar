import { create } from 'zustand'
import type { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/models.ts'
import {
  Recheck,
  Retry,
  States,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'

// Local, in-process reads of what the last real requests showed; the only network cost is the Recheck below.
const STATE_POLL_MS = 15_000
const RECHECK_MS = 5 * 60_000
// Go's zero time.
const NEVER_YEAR = 2000

const useOffline = create<{ states: State[]; set: (states: State[] | null) => void }>((set) => ({
  states: [],
  set: (states) => set({ states: states ?? [] }),
}))

function unreachable(states: State[], ids?: string[]): State[] {
  return states.filter((s) => s.unreachable && (!ids || ids.includes(s.id)))
}

export function savedAt(s: State, locale: string): string {
  const at = new Date(s.lastOK)
  return at.getFullYear() < NEVER_YEAR ? '' : at.toLocaleTimeString(locale, { timeStyle: 'short' })
}

function refresh(): Promise<void> {
  return States().then(useOffline.getState().set)
}

function retry(down: State[]): Promise<void> {
  return Promise.all(down.map((s) => Retry(s.id))).then(() => refresh())
}

function recheck(): Promise<void> {
  return Recheck().then(useOffline.getState().set)
}

export { RECHECK_MS, recheck, refresh, retry, STATE_POLL_MS, unreachable, useOffline }
