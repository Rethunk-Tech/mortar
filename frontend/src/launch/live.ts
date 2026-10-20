import { create } from 'zustand'
import {
  type LiveRun,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { idKey } from '../mods/dependents.ts'
import { sameValue } from '../shell/sameValue.ts'

interface Live {
  game: string
  profile: string
  // Whether each package that ships plugins has any of them loaded, by idKey of its mod id. The game's plugins are
  // a set, so this says nothing about load order.
  loaded: Record<string, boolean>
}

const EMPTY: Live = { game: '', profile: '', loaded: {} }

export const useLive = create<Live>(() => EMPTY)

export function applyLive(run: LiveRun) {
  const loaded: Record<string, boolean> = {}
  for (const m of run.mods ?? []) {
    loaded[idKey(m.id)] = m.loaded
  }
  const next = { game: run.game, profile: run.profile, loaded }
  if (!sameValue(useLive.getState(), next)) {
    useLive.setState(next)
  }
}

// A game that is no longer running has no live state to show.
export function clearLiveUnlessRunning(status: Pick<Status, 'game' | 'state'>) {
  if (status.game === useLive.getState().game && status.state !== State.Running) {
    useLive.setState(EMPTY)
  }
}
