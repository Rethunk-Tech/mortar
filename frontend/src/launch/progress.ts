import { create } from 'zustand'
import type { Progress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  State,
  type Status,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { LaunchProgress } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { waitsForLoader } from './stages.ts'

/**
 * The launch screen's phase: up while the game starts, then while its loader loads the mods, then a moment saying it
 * is running before it closes; a launch that fails stays up with the failure on it.
 */
export type Phase = 'closed' | 'launching' | 'loading' | 'running' | 'failed'

/** nextPhase is where the screen goes from phase when its launch reaches a state other than launching. */
export function nextPhase(phase: Phase, state: State, progress: Progress | null): Phase {
  if (phase === 'closed') {
    return phase
  }
  switch (state) {
    case State.Running:
      if (phase !== 'launching') {
        return phase
      }
      return progress && waitsForLoader(progress) ? 'loading' : 'running'
    case State.Failed:
      return phase === 'launching' || phase === 'loading' ? 'failed' : phase
    case State.Idle:
      // The game closed, or the launch was cancelled; a failure stays up until it is closed.
      return phase === 'failed' ? phase : 'closed'
    default:
      return 'closed'
  }
}

// A loader that stops reporting while it loads (it crashed, or never writes the line saying it is done) leaves the game
// running; the screen stops waiting after this long without news.
const LOADING_IDLE_MS = 60_000
let idle: ReturnType<typeof setTimeout> | undefined

interface LaunchView {
  game: string
  phase: Phase
  progress: Progress | null
  /** onStatus moves the screen on for a launch state; isNew is a launch that has just begun. */
  onStatus: (status: Status, isNew: boolean) => void
  onProgress: (progress: Progress) => void
  refresh: (game: string) => void
  close: () => void
}

export const useLaunchView = create<LaunchView>((set, get) => {
  // waitForNews restarts the wait for the loader's next line while the screen waits on it.
  const waitForNews = () => {
    clearTimeout(idle)
    if (get().phase === 'loading') {
      idle = setTimeout(() => {
        if (get().phase === 'loading') {
          set({ phase: 'running' })
        }
      }, LOADING_IDLE_MS)
    }
  }
  return {
    game: '',
    phase: 'closed',
    progress: null,
    onStatus: (status, isNew) => {
      const { phase, progress, game } = get()
      if (status.state === State.Launching) {
        if (isNew || phase === 'closed') {
          set({ game: status.game, phase: 'launching', progress: isNew ? null : progress })
        }
        return
      }
      if (game === status.game) {
        set({ phase: nextPhase(phase, status.state, progress) })
        waitForNews()
      }
    },
    onProgress: (next) => {
      const { game, phase } = get()
      if (next.game !== game) {
        return
      }
      set({
        progress: next,
        ...(phase === 'loading' && !waitsForLoader(next) ? { phase: 'running' } : {}),
      })
      waitForNews()
    },
    refresh: (game) => {
      LaunchProgress(game)
        .then((p) => {
          if (get().game === game && p.game === game) {
            set({ progress: p })
          }
        })
        .catch(reportUnexpected)
    },
    close: () => set({ phase: 'closed' }),
  }
})
