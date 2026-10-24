import { create } from 'zustand'
import type { State } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/models.ts'
import {
  Recheck,
  Retry,
  States,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/netstate/service.ts'
import { clockWhen } from '../i18n/when.ts'
import { shareInFlight } from '../profiles/listProfiles.ts'
import { sameValue } from './sameValue.ts'

// Local, in-process reads of what the last real requests showed; the only network cost is the Recheck below.
const STATE_POLL_MS = 15_000
const MINUTE_MS = 60_000
const RECHECK_MINUTES = 5
const RECHECK_MS = RECHECK_MINUTES * MINUTE_MS

const useOffline = create<{ states: State[]; set: (states: State[] | null) => void }>(
  (set, get) => ({
    states: [],
    set: (states) => {
      const next = states ?? []
      if (!sameValue(next, get().states)) {
        set({ states: next })
      }
    },
  }),
)

function unreachable(states: State[], ids?: string[]): State[] {
  return states.filter((s) => s.unreachable && (!ids || ids.includes(s.id)))
}

function savedAt(s: State, locale: string): string {
  return clockWhen(s.lastOK, locale)
}

const shared = shareInFlight<State[] | null>()

/** The sources' reachability. A state change reaches the banner and the Nexus details primer at once, so they share
 * one read. */
function readStates(): Promise<State[] | null> {
  return shared('', () => Promise.resolve(States()))
}

function refresh(): Promise<void> {
  return readStates().then(useOffline.getState().set)
}

function retry(down: State[]): Promise<void> {
  return Promise.all(down.map((s) => Retry(s.id))).then(() => refresh())
}

function recheck(): Promise<void> {
  return Recheck().then(useOffline.getState().set)
}

export {
  RECHECK_MS,
  readStates,
  recheck,
  refresh,
  retry,
  STATE_POLL_MS,
  savedAt,
  unreachable,
  useOffline,
}
