import { create } from 'zustand'
import {
  type LiveRun,
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { idKey } from '../mods/dependents.ts'

interface Live {
  game: string
  profile: string
  scene: string
  // Whether each package that ships plugins has any of them loaded, by idKey of its mod id. The game's plugins are
  // a set, so this says nothing about load order.
  loaded: Record<string, boolean>
}

const EMPTY: Live = { game: '', profile: '', scene: '', loaded: {} }

function same(a: Live, b: Live): boolean {
  return JSON.stringify(a) === JSON.stringify(b)
}

export const useLive = create<Live>(() => EMPTY)

export function applyLive(run: LiveRun) {
  const loaded: Record<string, boolean> = {}
  for (const m of run.mods ?? []) {
    loaded[idKey(m.id)] = m.loaded
  }
  const next = { game: run.game, profile: run.profile, scene: run.scene, loaded }
  if (!same(useLive.getState(), next)) {
    useLive.setState(next)
  }
}

// A game that is no longer running has no live state to show.
export function clearLiveUnlessRunning(status: Pick<Status, 'game' | 'state'>) {
  if (status.game === useLive.getState().game && status.state !== State.Running) {
    useLive.setState(EMPTY)
  }
}
